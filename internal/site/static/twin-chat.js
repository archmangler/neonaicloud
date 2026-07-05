(function () {
  "use strict";

  var root = document.getElementById("csuite-digital-twin");
  if (!root) {
    return;
  }

  var defaultPersona = root.dataset.persona || "cto";
  var showPicker = root.dataset.showPicker === "true";
  var healthBase = root.dataset.healthUrl || "/api/twin/health";
  var chatURL = root.dataset.chatUrl || "/api/twin/chat";
  var persona = defaultPersona;
  var histories = {};
  var personaMeta = {};
  var probedPersonas = {};

  var bodyEl = root.querySelector(".twin-console-body");
  var formEl = root.querySelector(".twin-composer");
  var inputEl = root.querySelector("#twin-message-input");
  var submitEl = root.querySelector(".twin-composer button[type='submit']");
  var noteEl = root.querySelector("#twin-composer-note");
  var statusChip = root.querySelector(".twin-console-meta .status-chip");
  var presenceLabel = root.querySelector(".twin-presence-label");
  var presenceDot = root.querySelector(".twin-presence-dot");
  var labelEl = root.querySelector(".twin-console-identity .card-title");
  var avatarEl = root.querySelector(".twin-console-identity .twin-avatar");
  var pickerEl = root.querySelector(".twin-persona-picker");
  var pickerButtons = pickerEl
    ? Array.prototype.slice.call(pickerEl.querySelectorAll(".twin-persona-chip"))
    : [];

  function personaFromQuery() {
    var params = new URLSearchParams(window.location.search);
    var value = (params.get("persona") || "").trim().toLowerCase();
    return value || null;
  }

  function personaConfig(id) {
    var button = pickerButtons.find(function (btn) {
      return btn.dataset.persona === id;
    });
    if (button) {
      return {
        id: id,
        initials: button.dataset.initials || id.toUpperCase(),
        label: button.dataset.label || id,
        intro: button.dataset.intro || "",
      };
    }
    return {
      id: id,
      initials: root.dataset.twinInitials || id.toUpperCase(),
      label: root.dataset.twinLabel || id,
      intro: root.dataset.twinIntro || "",
    };
  }

  function getHistory(id) {
    if (!histories[id]) {
      histories[id] = [];
    }
    return histories[id];
  }

  function escapeHTML(value) {
    return String(value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function formatParagraphs(text) {
    return escapeHTML(text)
      .split(/\n{2,}/)
      .map(function (block) {
        return "<p>" + block.replace(/\n/g, "<br>") + "</p>";
      })
      .join("");
  }

  function appendMessage(role, label, text, extraClass, options) {
    options = options || {};
    var config = personaConfig(persona);
    var wrap = document.createElement("div");
    wrap.className = "twin-message twin-message-" + role + (extraClass ? " " + extraClass : "");

    if (role === "system") {
      wrap.innerHTML =
        '<p class="twin-message-label">System</p>' +
        '<p class="twin-system-text">' + escapeHTML(text) + "</p>";
    } else {
      var initials = role === "user" ? "You" : config.initials;
      wrap.innerHTML =
        '<div class="twin-message-meta">' +
        '<span class="twin-avatar twin-avatar-sm" aria-hidden="true">' + escapeHTML(initials) + "</span>" +
        '<p class="twin-message-label">' + escapeHTML(label) + "</p>" +
        "</div>" +
        '<div class="twin-bubble">' +
        formatParagraphs(text) +
        "</div>";
    }

    bodyEl.appendChild(wrap);
    bodyEl.scrollTop = bodyEl.scrollHeight;

    if (!options.skipHistory && !extraClass) {
      getHistory(persona).push({ role: role, label: label, text: text });
    }

    return wrap;
  }

  function setComposerEnabled(enabled) {
    if (inputEl) {
      inputEl.disabled = !enabled;
    }
    if (submitEl) {
      submitEl.disabled = !enabled;
    }
    if (formEl) {
      formEl.setAttribute("aria-disabled", enabled ? "false" : "true");
    }
  }

  function setStatus(state, detail) {
    root.dataset.agentStatus = state;
    if (statusChip) {
      if (state === "connected") {
        statusChip.textContent = "Connected";
        statusChip.className = "chip status-chip status-published";
      } else if (state === "connecting") {
        statusChip.textContent = "Connecting";
        statusChip.className = "chip status-chip";
      } else {
        statusChip.textContent = "Standby";
        statusChip.className = "chip status-chip status-draft";
      }
    }
    if (presenceLabel) {
      presenceLabel.textContent =
        state === "connected" ? "Online" : state === "connecting" ? "Connecting" : "Standby";
    }
    if (presenceDot) {
      presenceDot.classList.toggle("twin-presence-dot-online", state === "connected");
    }
    if (noteEl && detail) {
      noteEl.textContent = detail;
    }
  }

  function setChipStatus(id, state) {
    var button = pickerButtons.find(function (btn) {
      return btn.dataset.persona === id;
    });
    if (!button) {
      return;
    }
    button.dataset.twinStatus = state;
    var statusEl = button.querySelector(".twin-chip-status");
    if (!statusEl) {
      return;
    }
    if (state === "online") {
      statusEl.textContent = "Online";
      statusEl.className = "twin-chip-status twin-chip-status-online";
    } else if (state === "connecting") {
      statusEl.textContent = "…";
      statusEl.className = "twin-chip-status";
    } else {
      statusEl.textContent = "Standby";
      statusEl.className = "twin-chip-status twin-chip-status-standby";
    }
  }

  function parseJSON(response) {
    return response.text().then(function (text) {
      if (!text) {
        return { ok: response.ok, data: {} };
      }
      try {
        return { ok: response.ok, data: JSON.parse(text) };
      } catch (parseErr) {
        var message = text;
        if (message.indexOf("Internal Server Error") === 0) {
          message = "The digital twin service returned an error. Check twin logs and LLM configuration.";
        }
        return { ok: false, data: { error: message } };
      }
    });
  }

  function errorFromData(data) {
    if (!data) {
      return "Request failed.";
    }
    if (typeof data.error === "string" && data.error) {
      return data.error;
    }
    if (typeof data.detail === "string" && data.detail) {
      return data.detail;
    }
    if (Array.isArray(data.detail)) {
      return data.detail.map(function (item) {
        return item.msg || JSON.stringify(item);
      }).join(" ");
    }
    return "Request failed.";
  }

  function healthURL(id) {
    return healthBase + "?persona=" + encodeURIComponent(id);
  }

  function renderHistory(id) {
    bodyEl.innerHTML = "";
    getHistory(id).forEach(function (message) {
      appendMessage(message.role, message.label, message.text, null, { skipHistory: true });
    });
    bodyEl.scrollTop = bodyEl.scrollHeight;
  }

  function updateIdentity(id) {
    var config = personaConfig(id);
    if (labelEl) {
      labelEl.textContent = personaMeta[id] && personaMeta[id].name ? personaMeta[id].name : config.label;
    }
    if (avatarEl) {
      avatarEl.textContent = config.initials;
    }
  }

  function updatePickerSelection(id) {
    pickerButtons.forEach(function (button) {
      var active = button.dataset.persona === id;
      button.classList.toggle("is-active", active);
      button.setAttribute("aria-selected", active ? "true" : "false");
    });
  }

  function applyConnectedState(id, data) {
    personaMeta[id] = {
      connected: true,
      name: data.name || personaConfig(id).label,
      status: "online",
    };
    setChipStatus(id, "online");
    if (id === persona) {
      updateIdentity(id);
      setStatus("connected", "Ask about capabilities, delivery, or engagement fit.");
      setComposerEnabled(true);
      if (inputEl) {
        inputEl.placeholder = "Message " + (data.name || personaConfig(id).label) + "…";
      }
    }
  }

  function applyStandbyState(id, message) {
    personaMeta[id] = {
      connected: false,
      name: personaConfig(id).label,
      status: "standby",
      error: message,
    };
    setChipStatus(id, "standby");
    if (id === persona) {
      setStatus("standby", message || "Digital twin is on standby. Use the enquiry form or try again later.");
      setComposerEnabled(false);
      if (inputEl) {
        inputEl.placeholder = "Digital twin on standby";
      }
    }
  }

  function probePersona(id) {
    setChipStatus(id, "connecting");
    return fetch(healthURL(id), {
      method: "GET",
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    })
      .then(function (response) {
        return parseJSON(response);
      })
      .then(function (result) {
        probedPersonas[id] = true;
        if (result.ok && result.data.status === "ok") {
          applyConnectedState(id, result.data);
          return true;
        }
        applyStandbyState(id, errorFromData(result.data) || "Digital twin is on standby.");
        return false;
      })
      .catch(function (err) {
        probedPersonas[id] = true;
        applyStandbyState(id, err.message || "Digital twin is on standby.");
        return false;
      });
  }

  function connect(id) {
    persona = id;
    root.dataset.persona = id;
    updateIdentity(id);
    updatePickerSelection(id);

    function renderIdleBody(online) {
      bodyEl.innerHTML = "";
      if (getHistory(id).length > 0) {
        renderHistory(id);
        return;
      }
      appendMessage("assistant", personaConfig(id).label, personaConfig(id).intro, null, {
        skipHistory: true,
      });
      if (!online) {
        appendMessage(
          "system",
          "System",
          (personaMeta[id] && personaMeta[id].error) ||
            "Digital twin is on standby. Use the enquiry form or try another twin.",
          null,
          { skipHistory: true }
        );
      }
    }

    if (personaMeta[id] && probedPersonas[id]) {
      renderIdleBody(personaMeta[id].connected);
      if (personaMeta[id].connected) {
        updateIdentity(id);
        setStatus("connected", "Ask about capabilities, delivery, or engagement fit.");
        setComposerEnabled(true);
        if (inputEl) {
          inputEl.placeholder = "Message " + personaMeta[id].name + "…";
        }
      } else {
        setStatus("standby", personaMeta[id].error || "Digital twin is on standby.");
        setComposerEnabled(false);
        if (inputEl) {
          inputEl.placeholder = "Digital twin on standby";
        }
      }
      return;
    }

    setStatus("connecting", "Checking digital twin availability…");
    setComposerEnabled(false);
    renderIdleBody(false);

    probePersona(id).then(function (online) {
      if (id !== persona) {
        return;
      }
      renderIdleBody(online);
    });
  }

  function switchPersona(id) {
    if (id === persona) {
      return;
    }
    connect(id);
    if (showPicker) {
      var url = new URL(window.location.href);
      url.searchParams.set("persona", id);
      history.replaceState(null, "", url.pathname + url.search + url.hash);
    }
  }

  function sendMessage(message) {
    if (!personaMeta[persona] || !personaMeta[persona].connected) {
      return;
    }

    setComposerEnabled(false);
    appendMessage("user", "You", message);

    var pending = appendMessage("assistant", personaConfig(persona).label, "Thinking…", "twin-message-pending", {
      skipHistory: true,
    });

    fetch(chatURL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "same-origin",
      body: JSON.stringify({
        persona: persona,
        message: message,
        history: getHistory(persona)
          .filter(function (item) {
            return item.role === "user" || item.role === "assistant";
          })
          .slice(0, -1)
          .map(function (item) {
            return { role: item.role, content: item.text };
          }),
      }),
    })
      .then(function (response) {
        return parseJSON(response);
      })
      .then(function (result) {
        pending.remove();
        if (!result.ok) {
          throw new Error(errorFromData(result.data) || "Chat request failed.");
        }

        var reply = result.data.reply || "";
        var label = result.data.name || personaConfig(persona).label;
        appendMessage("assistant", label, reply);
        setComposerEnabled(true);
        if (inputEl) {
          inputEl.focus();
        }
      })
      .catch(function (err) {
        pending.remove();
        appendMessage("system", "System", err.message || "Something went wrong. Try again or use the enquiry form.");
        getHistory(persona).pop();
        setComposerEnabled(true);
        if (inputEl) {
          inputEl.focus();
        }
      });
  }

  pickerButtons.forEach(function (button) {
    button.addEventListener("click", function () {
      switchPersona(button.dataset.persona);
    });
  });

  if (formEl) {
    formEl.addEventListener("submit", function (event) {
      event.preventDefault();
      if (!inputEl || inputEl.disabled) {
        return;
      }
      var message = inputEl.value.trim();
      if (!message) {
        return;
      }
      inputEl.value = "";
      sendMessage(message);
    });
  }

  var initialPersona = personaFromQuery() || defaultPersona;
  if (showPicker) {
    var known = pickerButtons.some(function (button) {
      return button.dataset.persona === initialPersona;
    });
    if (!known) {
      initialPersona = defaultPersona;
    }
    Promise.all(
      pickerButtons.map(function (button) {
        return probePersona(button.dataset.persona);
      })
    ).then(function () {
      connect(initialPersona);
    });
  } else {
    probePersona(initialPersona).then(function () {
      connect(initialPersona);
    });
  }
})();
