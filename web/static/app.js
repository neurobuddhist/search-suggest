const searchInput = document.querySelector("#search");
const engineSelect = document.querySelector("#engine");
const kInput = document.querySelector("#k");
const kValue = document.querySelector("#k-value");
const results = document.querySelector("#results");
const empty = document.querySelector("#empty");
const visibleEngines = new Set(["linear", "sorted", "radix", "ranked-trie"]);

let abortController;
let debounceTimer;
let requestID = 0;

async function loadEngines() {
  const response = await fetch("/api/engines");
  if (!response.ok) {
    throw new Error("failed to load engines");
  }
  const data = await response.json();
  if (!Array.isArray(data.engines)) {
    throw new Error("invalid engine response");
  }

  const engines = data.engines.filter((engine) => visibleEngines.has(engine));

  engineSelect.replaceChildren(
    ...engines.map((engine) => {
      const option = document.createElement("option");
      option.value = engine;
      option.textContent = engine;
      if (engine === "ranked-trie") {
        option.selected = true;
      }
      return option;
    }),
  );
}

function scheduleSuggest() {
  window.clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(fetchSuggestions, 120);
}

async function fetchSuggestions() {
  abortController?.abort();
  const currentRequestID = ++requestID;
  abortController = new AbortController();

  const params = new URLSearchParams({
    text: searchInput.value,
    engine: engineSelect.value,
    k: kInput.value,
  });

  try {
    const response = await fetch(`/api/suggest?${params}`, {
      signal: abortController.signal,
    });
    const data = await response.json();
    if (currentRequestID !== requestID) {
      return;
    }
    if (!response.ok) {
      throw new Error(data.error || "request failed");
    }
    render(data);
  } catch (error) {
    if (error.name !== "AbortError" && currentRequestID === requestID) {
      render({ suggestions: [] });
    }
  }
}

function render(data) {
  const suggestions = Array.isArray(data.suggestions) ? data.suggestions : [];
  results.replaceChildren(
    ...suggestions.map((item) => {
      const row = document.createElement("li");

      const phrase = document.createElement("span");
      phrase.className = "phrase";
      phrase.textContent = item.text;

      const score = document.createElement("span");
      score.className = "score";
      score.textContent = item.score;

      row.append(phrase, score);
      return row;
    }),
  );
  empty.hidden = suggestions.length > 0;
}

kInput.addEventListener("input", () => {
  kValue.textContent = kInput.value;
  scheduleSuggest();
});
searchInput.addEventListener("input", scheduleSuggest);
engineSelect.addEventListener("change", fetchSuggestions);

loadEngines()
  .then(fetchSuggestions)
  .catch(() => render({ suggestions: [] }));
