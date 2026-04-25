const searchInput = document.querySelector("#search");
const engineSelect = document.querySelector("#engine");
const kInput = document.querySelector("#k");
const kValue = document.querySelector("#k-value");
const results = document.querySelector("#results");
const empty = document.querySelector("#empty");

let abortController;
let debounceTimer;

async function loadEngines() {
  const response = await fetch("/api/engines");
  const data = await response.json();

  engineSelect.replaceChildren(
    ...data.engines.map((engine) => {
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
    if (!response.ok) {
      throw new Error(data.error || "request failed");
    }
    render(data);
  } catch (error) {
    if (error.name !== "AbortError") {
      render({ suggestions: [] });
    }
  }
}

function render(data) {
  results.replaceChildren(
    ...data.suggestions.map((item) => {
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
  empty.hidden = data.suggestions.length > 0;
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
