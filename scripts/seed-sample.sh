#!/bin/bash
set -euo pipefail

# seed-sample.sh — Create 50 well-known books via BFF API
# Usage: ./seed-sample.sh [BFF_URL]

BFF_URL="${BFF_URL:-http://localhost:8010}"
SEED_EMAIL="${SEED_EMAIL:-seed@verso.app}"
SEED_PASSWORD="${SEED_PASSWORD:-SeedUser1234!}"
WAIT_SECS="${WAIT_SECS:-15}"

echo "=== Verso Sample Seed (50 books) ==="
echo "BFF: $BFF_URL"
echo ""

# ── Auth ────────────────────────────────────────────────────────
echo "--- Registering seed user ---"
REGISTER_RESP=$(curl -sf -X POST "$BFF_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$SEED_EMAIL\",\"password\":\"$SEED_PASSWORD\",\"displayName\":\"Seed User\"}" 2>/dev/null || true)

if [ -z "$REGISTER_RESP" ] || [ "$(echo "$REGISTER_RESP" | jq -r '.accessToken // empty' 2>/dev/null)" = "" ]; then
  echo "  Registration failed or user exists. Attempting login..."
  REGISTER_RESP=$(curl -sf -X POST "$BFF_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$SEED_EMAIL\",\"password\":\"$SEED_PASSWORD\"}")
fi

TOKEN=$(echo "$REGISTER_RESP" | jq -r '.accessToken')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "ERROR: Could not obtain auth token"
  exit 1
fi
echo "  Token: ${TOKEN:0:30}..."
echo ""

# ── Book data (50 well-known works) ─────────────────────────────
# Format: title|year|description
BOOKS=(
  'Dune|1965|A science fiction masterpiece about politics, religion, and ecology on the desert planet Arrakis.'
  '1984|1949|A dystopian novel depicting a totalitarian society under the omnipresent surveillance of Big Brother.'
  'The Great Gatsby|1925|A portrait of the Jazz Age and the American Dream through the eyes of Nick Carraway.'
  'To Kill a Mockingbird|1960|A young girl in the American South witnesses her father defend a Black man accused of a terrible crime.'
  'Neuromancer|1984|A washed-up computer hacker is hired for one last job in this pioneering cyberpunk novel.'
  'Snow Crash|1992|A novel blending Sumerian mythology, computer science, and pizza delivery in a privatized America.'
  "The Hitchhiker's Guide to the Galaxy|1979|Arthur Dent is whisked off Earth moments before its demolition to make way for a hyperspace bypass."
  'The Lord of the Rings|1954|An epic high-fantasy quest to destroy the One Ring and defeat the Dark Lord Sauron.'
  "Harry Potter and the Philosopher's Stone|1997|An orphaned boy discovers he is a wizard and begins his education at Hogwarts School."
  'Pride and Prejudice|1813|A witty romance between the headstrong Elizabeth Bennet and the proud Mr. Darcy in Regency England.'
  'Brave New World|1932|A dystopian society achieves stability through technological and social engineering at the cost of freedom.'
  'The Catcher in the Rye|1951|Holden Caulfield narrates his experiences in New York City after being expelled from prep school.'
  'Fahrenheit 451|1953|In a future where books are banned, a fireman begins to question his role in burning them.'
  'One Hundred Years of Solitude|1967|Seven generations of the Buendía family in the fictional town of Macondo.'
  'Slaughterhouse-Five|1969|Billy Pilgrim becomes unstuck in time, traveling between World War II and an alien planet.'
  'The Handmaid'"'"'s Tale|1985|In the theocratic Republic of Gilead, women are stripped of rights and forced into servitude.'
  'Beloved|1987|A formerly enslaved woman is haunted by the ghost of her daughter in post-Civil War Ohio.'
  'The Road|2006|A father and son journey through a post-apocalyptic landscape, surviving on love and hope.'
  'Frankenstein|1818|A young scientist creates a sapient creature in an unorthodox experiment, with devastating consequences.'
  'Dracula|1897|An ancient vampire from Transylvania moves to England to spread the undead curse.'
  'Jane Eyre|1847|An orphaned governess falls in love with her employer, the brooding Mr. Rochester.'
  'Wuthering Heights|1847|A tale of passionate and destructive love between Heathcliff and Catherine on the Yorkshire moors.'
  'Moby-Dick|1851|Captain Ahab obsessively pursues the great white whale across the oceans.'
  'War and Peace|1869|Five aristocratic families navigate Russian society during the Napoleonic Wars.'
  'Crime and Punishment|1866|A destitute student murders a pawnbroker and grapples with guilt and morality in St. Petersburg.'
  'The Brothers Karamazov|1880|Three brothers and their dissolute father confront questions of faith, doubt, and morality.'
  'Anna Karenina|1878|A married aristocrat begins a passionate affair that leads to her social and personal downfall.'
  'Don Quixote|1605|An aging nobleman loses his sanity and decides to become a knight-errant to revive chivalry.'
  'The Odyssey|0800|Odysseus endures a decade-long journey home after the fall of Troy, facing monsters and gods.'
  'The Divine Comedy|1320|Dante journeys through Hell, Purgatory, and Paradise guided by Virgil and Beatrice.'
  'Les Misérables|1862|The story of ex-convict Jean Valjean and his quest for redemption in 19th-century France.'
  'The Count of Monte Cristo|1844|A wrongly imprisoned man escapes and uses a hidden treasure to exact revenge on those who betrayed him.'
  'A Tale of Two Cities|1859|A novel set in London and Paris before and during the French Revolution.'
  'Great Expectations|1861|An orphan named Pip navigates class and ambition in Victorian England.'
  'The Picture of Dorian Gray|1890|A young man remains beautiful while his portrait ages and reflects his moral corruption.'
  'Heart of Darkness|1899|A voyage up the Congo River into the interior of Africa becomes a journey into the darkness of the human soul.'
  'Catch-22|1961|A World War II bombardier is trapped by the absurd bureaucratic rule called Catch-22.'
  'The Grapes of Wrath|1939|The Joad family travels from Oklahoma to California during the Great Depression.'
  'On the Road|1957|Sal Paradise and Dean Moriarty drive back and forth across America in search of meaning.'
  'Invisible Man|1952|An unnamed Black man recounts his journey from the Deep South to Harlem, confronting racism and identity.'
  'The Color Purple|1982|Celie writes letters to God chronicling her life of abuse and her path toward empowerment.'
  'Lolita|1955|An unreliable narrator describes his obsession with a twelve-year-old girl in postwar America.'
  'The Sun Also Rises|1926|A group of American and British expatriates travel from Paris to Pamplona for the running of the bulls.'
  'A Clockwork Orange|1962|In a near-future Britain, a young delinquent undergoes experimental conditioning to cure his violent behavior.'
  'The Left Hand of Darkness|1969|An envoy visits a planet whose inhabitants can change their biological sex, challenging assumptions about gender.'
  'Foundation|1951|A mathematician predicts the fall of the Galactic Empire and establishes a foundation to shorten the coming dark age.'
  'Do Androids Dream of Electric Sheep?|1968|A bounty hunter pursues rogue androids in a post-apocalyptic San Francisco.'
  'The Name of the Rose|1980|A Franciscan friar investigates a series of murders in a medieval Italian abbey.'
  'One Flew Over the Cuckoo'"'"'s Nest|1962|A convict disrupts the routine of a mental institution, challenging its authoritarian nurse.'
  'Things Fall Apart|1958|An Igbo leader watches his world collapse under the pressure of British colonialism in Nigeria.'
)

# ── Create works ────────────────────────────────────────────────
SUCCESS=0
FAILED=0
DUPES=0

echo "--- Creating 50 works ---"
for entry in "${BOOKS[@]}"; do
  IFS='|' read -r TITLE YEAR DESC <<< "$entry"

  # Build JSON payload
  PAYLOAD=$(jq -n \
    --arg title "$TITLE" \
    --arg desc "$DESC" \
    --argjson year "$YEAR" \
    '{title: $title, description: $desc, originalLanguage: "en", originalPublicationYear: $year}')

  HTTP_CODE=$(curl -s -o /tmp/verso_seed_resp.json -w "%{http_code}" \
    -X POST "$BFF_URL/api/v1/books" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD")

  if [ "$HTTP_CODE" = "201" ]; then
    WORK_ID=$(jq -r '.id' /tmp/verso_seed_resp.json 2>/dev/null || echo "unknown")
    echo "  ✓ $TITLE ($YEAR) → $WORK_ID"
    SUCCESS=$((SUCCESS + 1))
  elif [ "$HTTP_CODE" = "409" ]; then
    echo "  ⊘ $TITLE — already exists (skipped)"
    DUPES=$((DUPES + 1))
  else
    echo "  ✗ $TITLE — HTTP $HTTP_CODE"
    FAILED=$((FAILED + 1))
  fi
done

rm -f /tmp/verso_seed_resp.json
echo ""
echo "--- Results: $SUCCESS created, $DUPES duplicates, $FAILED failed ---"
echo ""

# ── Wait for search indexer ─────────────────────────────────────
echo "--- Waiting ${WAIT_SECS}s for search indexer ---"
sleep "$WAIT_SECS"

# ── Verify search ───────────────────────────────────────────────
echo "--- Verifying search ---"
SEARCH_RESP=$(curl -sf "$BFF_URL/api/v1/search?q=dune&type=work" 2>/dev/null || echo '{"results":[]}')
HITS=$(echo "$SEARCH_RESP" | jq '.results | length' 2>/dev/null || echo "0")
echo "  Search 'dune': $HITS result(s)"

SEARCH_RESP2=$(curl -sf "$BFF_URL/api/v1/search?q=gatsby&type=work" 2>/dev/null || echo '{"results":[]}')
HITS2=$(echo "$SEARCH_RESP2" | jq '.results | length' 2>/dev/null || echo "0")
echo "  Search 'gatsby': $HITS2 result(s)"

echo ""
if [ "$SUCCESS" -gt 0 ] || [ "$DUPES" -gt 0 ]; then
  TOTAL=$((SUCCESS + DUPES))
  echo "=== Seeded $TOTAL works successfully ==="
else
  echo "=== SEED FAILED — no works created ==="
  exit 1
fi
