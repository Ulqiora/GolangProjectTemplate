const state = {
  session: readSession(),
};

const els = {
  tabs: document.querySelectorAll(".tab"),
  forms: document.querySelectorAll(".form"),
  loginForm: document.querySelector("#login-form"),
  registerForm: document.querySelector("#register-form"),
  yandex: document.querySelector("#yandex-login"),
  message: document.querySelector("#message"),
  logout: document.querySelector("#logout"),
  sessionTitle: document.querySelector("#session-title"),
  profileCard: document.querySelector("#profile-card"),
  tokenPreview: document.querySelector("#token-preview"),
  animeSearch: document.querySelector("#anime-search"),
  animeGrid: document.querySelector("#anime-grid"),
  animeStatus: document.querySelector("#anime-status"),
  animeFilters: document.querySelectorAll(".filter"),
};

for (const tab of els.tabs) {
  tab.addEventListener("click", () => setView(tab.dataset.view));
}

els.loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = formData(event.currentTarget);
  await authenticate(() => api("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({
      identifier: data.identifier,
      password: data.password,
    }),
  }));
});

els.registerForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = formData(event.currentTarget);
  await authenticate(() => api("/v1/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
  }));
});

els.yandex.addEventListener("click", async () => {
  setMessage("Готовим переход в Яндекс ID...");
  setBusy(els.yandex, true);
  try {
    const result = await api("/v1/auth/yandex/url");
    const url = result.authorizationUrl || result.authorization_url;
    if (!url) {
      throw new Error("API не вернул authorization_url");
    }
    window.location.assign(url);
  } catch (error) {
    setMessage(error.message, "error");
  } finally {
    setBusy(els.yandex, false);
  }
});

els.logout.addEventListener("click", () => {
  state.session = null;
  localStorage.removeItem("auth.session");
  renderSession();
  setMessage("Вы вышли из аккаунта.");
});

els.animeSearch.addEventListener("submit", async (event) => {
  event.preventDefault();
  const query = new FormData(event.currentTarget).get("query")?.trim();
  if (!query) {
    await loadAnime("top");
    return;
  }
  setAnimeFilter("");
  await loadAnime("search", query);
});

for (const filter of els.animeFilters) {
  filter.addEventListener("click", async () => {
    setAnimeFilter(filter.dataset.mode);
    els.animeSearch.reset();
    await loadAnime(filter.dataset.mode);
  });
}

completeYandexCallback();
renderSession();
loadAnime("top");

function setView(view) {
  for (const tab of els.tabs) {
    tab.classList.toggle("active", tab.dataset.view === view);
  }
  for (const form of els.forms) {
    form.classList.toggle("active", form.dataset.view === view);
  }
  setMessage("");
}

async function authenticate(request) {
  setMessage("Отправляем запрос...");
  const button = document.activeElement?.tagName === "BUTTON" ? document.activeElement : null;
  setBusy(button, true);
  try {
    const result = await request();
    saveSession(result);
    setMessage("Готово. Токены сохранены локально.", "success");
  } catch (error) {
    setMessage(error.message, "error");
  } finally {
    setBusy(button, false);
  }
}

async function completeYandexCallback() {
  const url = new URL(window.location.href);
  const code = url.searchParams.get("code");
  const oauthState = url.searchParams.get("state");
  const error = url.searchParams.get("error");

  if (error) {
    setMessage(`Яндекс ID вернул ошибку: ${error}`, "error");
    window.history.replaceState({}, "", "/");
    return;
  }
  if (!code || !oauthState) {
    return;
  }

  setMessage("Завершаем вход через Яндекс ID...");
  try {
    const result = await api("/v1/auth/yandex/complete", {
      method: "POST",
      body: JSON.stringify({ code, state: oauthState }),
    });
    saveSession(result);
    setMessage("Вход через Яндекс ID выполнен.", "success");
  } catch (callbackError) {
    setMessage(callbackError.message, "error");
  } finally {
    window.history.replaceState({}, "", "/");
  }
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...options.headers,
    },
    ...options,
  });

  const text = await response.text();
  const payload = text ? JSON.parse(text) : {};
  if (!response.ok) {
    throw new Error(payload.message || payload.error || `HTTP ${response.status}`);
  }
  return payload;
}

function saveSession(payload) {
  state.session = {
    userId: payload.userId || payload.user_id || "",
    accessToken: payload.accessToken || payload.access_token || "",
    refreshToken: payload.refreshToken || payload.refresh_token || "",
    provider: payload.provider || "password",
    createdAt: payload.createdAt || payload.created_at || new Date().toISOString(),
  };
  localStorage.setItem("auth.session", JSON.stringify(state.session));
  renderSession();
}

function readSession() {
  try {
    const raw = localStorage.getItem("auth.session");
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function renderSession() {
  const session = state.session;
  els.logout.classList.toggle("hidden", !session);
  if (!session) {
    els.sessionTitle.textContent = "Нет активного входа";
    els.profileCard.innerHTML = '<div class="empty-state">Войдите через пароль или Яндекс ID, чтобы увидеть данные сессии.</div>';
    els.tokenPreview.textContent = "-";
    return;
  }

  els.sessionTitle.textContent = "Активная сессия";
  els.profileCard.innerHTML = `
    <div class="profile-grid">
      ${row("User ID", session.userId)}
      ${row("Провайдер", session.provider)}
      ${row("Создано", formatDate(session.createdAt))}
      ${row("Refresh token", session.refreshToken ? mask(session.refreshToken) : "-")}
    </div>
  `;
  els.tokenPreview.textContent = session.accessToken || "-";
}

function row(label, value) {
  return `
    <div class="profile-row">
      <div class="profile-label">${escapeHtml(label)}</div>
      <div class="profile-value">${escapeHtml(value || "-")}</div>
    </div>
  `;
}

function formData(form) {
  return Object.fromEntries(new FormData(form).entries());
}

function setMessage(text, type = "") {
  els.message.textContent = text;
  els.message.className = `message ${type}`.trim();
}

function setBusy(button, isBusy) {
  if (!button) {
    return;
  }
  button.disabled = isBusy;
  button.dataset.text ??= button.textContent;
  button.textContent = isBusy ? "Подождите..." : button.dataset.text;
}

function formatDate(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function mask(value) {
  if (value.length <= 16) {
    return value;
  }
  return `${value.slice(0, 8)}...${value.slice(-8)}`;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

async function loadAnime(mode, query = "") {
  const endpoints = {
    top: "/v1/anime/catalog?mode=top",
    airing: "/v1/anime/catalog?mode=airing",
    upcoming: "/v1/anime/catalog?mode=upcoming",
    search: `/v1/anime/catalog?mode=search&q=${encodeURIComponent(query)}`,
  };

  setAnimeStatus(mode === "search" ? `Ищем: ${query}...` : "Загружаем каталог...");
  els.animeGrid.innerHTML = skeletonCards();

  try {
    const response = await fetch(endpoints[mode], { headers: { Accept: "application/json" } });
    if (!response.ok) {
      throw new Error(`Jikan API вернул HTTP ${response.status}`);
    }
    const payload = await response.json();
    const items = Array.isArray(payload.data) ? payload.data : [];
    if (items.length === 0) {
      els.animeGrid.innerHTML = "";
      setAnimeStatus("Ничего не найдено.");
      return;
    }
    els.animeGrid.innerHTML = items.map(animeCard).join("");
    setAnimeStatus(mode === "search" ? `Найдено: ${items.length}` : "Популярные тайтлы обновлены.");
  } catch (error) {
    els.animeGrid.innerHTML = "";
    setAnimeStatus(error.message, "error");
  }
}

function setAnimeFilter(mode) {
  for (const filter of els.animeFilters) {
    filter.classList.toggle("active", filter.dataset.mode === mode);
  }
}

function setAnimeStatus(text, type = "") {
  els.animeStatus.textContent = text;
  els.animeStatus.className = `anime-status ${type}`.trim();
}

function animeCard(item) {
  const image = item.images?.webp?.large_image_url || item.images?.jpg?.large_image_url || "";
  const title = item.title_russian || item.title_english || item.title || "Без названия";
  const type = item.type || "TV";
  const episodes = item.episodes ? `${item.episodes} эп.` : "эпизоды неизвестны";
  const year = item.year || item.aired?.prop?.from?.year || "год неизвестен";
  const score = item.score ? item.score.toFixed(1) : "-";
  const synopsis = truncate(item.synopsis || "Описание пока не добавлено.", 138);
  const url = item.url || "#";

  return `
    <article class="anime-card">
      <div class="anime-poster">
        ${image ? `<img src="${escapeHtml(image)}" alt="${escapeHtml(title)}" loading="lazy">` : ""}
        <span class="anime-score">${escapeHtml(score)}</span>
      </div>
      <div class="anime-body">
        <h3 class="anime-title">${escapeHtml(title)}</h3>
        <div class="anime-meta">
          <span>${escapeHtml(type)}</span>
          <span>${escapeHtml(episodes)}</span>
          <span>${escapeHtml(year)}</span>
        </div>
        <p class="anime-synopsis">${escapeHtml(synopsis)}</p>
        <a class="anime-link" href="${escapeHtml(url)}" target="_blank" rel="noreferrer">Открыть на MyAnimeList</a>
      </div>
    </article>
  `;
}

function skeletonCards() {
  return Array.from({ length: 4 }, (_, index) => `
    <article class="anime-card" aria-hidden="true">
      <div class="anime-poster"></div>
      <div class="anime-body">
        <h3 class="anime-title">Загрузка ${index + 1}</h3>
        <div class="anime-meta"><span>...</span><span>...</span></div>
        <p class="anime-synopsis">Получаем данные из каталога.</p>
      </div>
    </article>
  `).join("");
}

function truncate(value, maxLength) {
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, maxLength - 1).trim()}...`;
}
