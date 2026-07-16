import type { FastifyInstance, FastifyRequest, FastifyReply } from "fastify";
import { z } from "zod";
import { ulid } from "ulid";
import type pg from "pg";
import type CircuitBreaker from "opossum";
import type { UploadParams } from "../lib/s3.js";
import { requireAuth } from "../middleware/auth.js";
import type { Config } from "../config.js";

const ALLOWED_MIME_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
]);

const entityTypeSchema = z.enum(["cover", "avatar", "attachment"]);
const ulidSchema = z.string().length(26).optional();

interface MediaDeps {
  config: Config;
  pool: pg.Pool;
  uploadBreaker: CircuitBreaker<[UploadParams], void>;
  presignBreaker: CircuitBreaker<[string, string, number], string>;
}

export async function mediaRoutes(
  app: FastifyInstance,
  deps: MediaDeps,
): Promise<void> {
  const { config, pool, uploadBreaker, presignBreaker } = deps;

  app.post(
    "/v1/media/upload",
    { preHandler: requireAuth },
    async (request: FastifyRequest, reply: FastifyReply) => {
      const userId = (request as FastifyRequest & { userId: string }).userId;
      const file = await request.file({
        limits: { fileSize: config.maxFileSize },
      });

      if (!file) {
        return reply.code(400).send({
          error: "bad_request",
          message: "No file provided",
        });
      }

      if (!ALLOWED_MIME_TYPES.has(file.mimetype)) {
        return reply.code(400).send({
          error: "bad_request",
          message: `Unsupported mime type: ${file.mimetype}. Allowed: ${[...ALLOWED_MIME_TYPES].join(", ")}`,
        });
      }

      const fields = file.fields as Record<
        string,
        { value?: string } | undefined
      >;
      const entityTypeRaw = fields["entityType"]?.value;
      const entityIdRaw = fields["entityId"]?.value;

      const entityTypeParsed = entityTypeSchema.safeParse(entityTypeRaw);
      if (!entityTypeParsed.success) {
        return reply.code(400).send({
          error: "bad_request",
          message:
            "entityType is required and must be one of: cover, avatar, attachment",
        });
      }
      const entityType = entityTypeParsed.data;

      const entityIdParsed = ulidSchema.safeParse(entityIdRaw || undefined);
      if (!entityIdParsed.success) {
        return reply.code(400).send({
          error: "bad_request",
          message: "entityId must be a valid 26-character ULID",
        });
      }
      const entityId = entityIdParsed.data ?? null;

      let buffer: Buffer;
      try {
        buffer = await file.toBuffer();
      } catch (err: unknown) {
        const code =
          err && typeof err === "object" && "code" in err
            ? (err as { code: string }).code
            : "";
        if (code === "FST_REQ_FILE_TOO_LARGE") {
          return reply.code(413).send({
            error: "payload_too_large",
            message: `File exceeds maximum size of ${config.maxFileSize} bytes`,
          });
        }
        throw err;
      }

      if (file.file.truncated) {
        return reply.code(413).send({
          error: "payload_too_large",
          message: `File exceeds maximum size of ${config.maxFileSize} bytes`,
        });
      }

      const assetId = ulid();
      const objectKey = `${entityType}/${assetId}/${file.filename}`;

      await uploadBreaker.fire({
        bucket: config.minio.bucket,
        key: objectKey,
        body: buffer,
        contentType: file.mimetype,
      });

      await pool.query(
        `INSERT INTO media.media_asset
           (id, uploader_id, file_name, mime_type, file_size, object_key, bucket, entity_type, entity_id, upload_status)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'completed')`,
        [
          assetId,
          userId,
          file.filename,
          file.mimetype,
          buffer.length,
          objectKey,
          config.minio.bucket,
          entityType,
          entityId,
        ],
      );

      const url = await presignBreaker.fire(
        config.minio.bucket,
        objectKey,
        config.presignedUrlExpiry,
      );

      return reply.code(201).send({
        id: assetId,
        fileName: file.filename,
        mimeType: file.mimetype,
        fileSize: buffer.length,
        url,
      });
    },
  );

  app.get(
    "/v1/media/:id",
    async (
      request: FastifyRequest<{ Params: { id: string } }>,
      reply: FastifyReply,
    ) => {
      const { id } = request.params;
      const result = await pool.query(
        "SELECT object_key, bucket, file_name, mime_type, file_size FROM media.media_asset WHERE id = $1",
        [id],
      );

      if (result.rows.length === 0) {
        return reply.code(404).send({
          error: "not_found",
          message: "Media asset not found",
        });
      }

      const row = result.rows[0];
      const url = await presignBreaker.fire(
        row.bucket,
        row.object_key,
        config.presignedUrlExpiry,
      );

      return {
        id,
        fileName: row.file_name,
        mimeType: row.mime_type,
        fileSize: Number(row.file_size),
        url,
      };
    },
  );
}
