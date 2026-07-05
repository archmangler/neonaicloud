(function () {
  "use strict";

  var root = document.getElementById("csuite-digital-twin");
  if (!root) {
    return;
  }

  var persona = root.dataset.persona || "cto";
  var healthURL = root.dataset.healthUrl || "/api/twin/health?persona=" + encodeURIComponent(persona);
  var chatURL = root.dataset.chatUrl || "/api/twin/chat";
  var history = [];

  var bodyEl = root.querySelector(".twin-console-body");
  var formEl = root.querySelector(".twin-composer");
  var inputEl = root.querySelector("#twin-message-input");
  var submitEl = root.querySelector(".twin-composer button[type='submit']");
  var noteEl = root.querySelector("#twin-composer-note");
  var statusChip = root.querySelector(".twin-console-meta .status-chip");
  var presenceLabel = root.querySelector(".twin-presence-label");
  var presenceDot = root.querySelector(".twin-presence-dot");
  var labelEl = root.querySelector(".twin-console-identity .card-title");

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

  function appendMessage(role, label, text, extraClass) {
    var wrap = document.createElement("div");
    wrap.className = "twin-message twin-message-" + role + (extraClass ? " " + extraClass : "");

    if (role === "system") {
      wrap.innerHTML =
        '<p class="twin-message-label">System</p>' +
        '<p class="twin-system-text">' + escapeHTML(text) + "</p>";
    } else {
      var initials = role === "user" ? "You" : "CTO";
      wrap.innerHTML =
        '<div class="twin-message-meta">' +
        '<span class="twin-avatar twin-avatar-sm" aria-hidden="true">' + initials + "</span>" +
        '<p class="twin-message-label">' + escapeHTML(label) + "</p>" +
        "</div>" +
        '<div class="twin-bubble">' +
        formatParagraphs(text) +
        "</div>";
    }

    bodyEl.appendChild(wrap);
    bodyEl.scrollTop = bodyEl.scrollHeight;
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
        statusChip.textContent = "Unavailable";
        statusChip.className = "chip status-chip status-draft";
      }
    }
    if (presenceLabel) {
      presenceLabel.textContent = state === "connected" ? "Online" : state === "connecting" ? "Connecting" : "Standby";
    }
    if (presenceDot) {
      presenceDot.classList.toggle("twin-presence-dot-online", state === "connected");
    }
    if (noteEl && detail) {
      noteEl.textContent = detail;
    }
  }

  function removeStaticMessages() {
    root.querySelectorAll('[data-static="true"]').forEach(function (node) {
      node.remove();
    });
  }

  function connect() {
    setStatus("connecting", "Checking digital twin availability…");
    setComposerEnabled(false);

    fetch(healthURL, {
      method: "GET",
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    })
      .then(function (response) {
        return response.json().then(function (data) {
          return { ok: response.ok, data: data };
        });
      })
      .then(function (result) {
        if (!result.ok || result.data.status !== "ok") {
          throw new Error(result.data.error || "Digital twin is unavailable.");
        }

        removeStaticMessages();
        if (labelEl && result.data.name) {
          labelEl.textContent = result.data.name;
        }

        setStatus("connected", "Ask about capabilities, delivery, or engagement fit.");
        setComposerEnabled(true);
        if (inputEl) {
          inputEl.placeholder = "Message " + (result.data.name || "the digital twin") + "…";
          inputEl.focus();
        }
      })
      .catch(function (err) {
        setStatus("unavailable", err.message || "Digital twin is unavailable. Use the enquiry form.");
        setComposerEnabled(false);
      });
  }

  function sendMessage(message) {
    setComposerEnabled(false);
    appendMessage("user", "You", message);
    history.push({ role: "user", content: message });

    var pending = appendMessage("assistant", "Digital twin", "Thinking…", "twin-message-pending");

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
        history: history.slice(0, -1),
      }),
    })
      .then(function (response) {
        return response.json().then(function (data) {
          return { ok: response.ok, data: data };
        });
      })
      .then(function (result) {
        pending.remove();
        if (!result.ok) {
          throw new Error(result.data.error || result.data.detail || "Chat request failed.");
        }

        var reply = result.data.reply || "";
        var label = result.data.name || "Digital twin";
        appendMessage("assistant", label, reply);
        history.push({ role: "assistant", content: reply });
        setComposerEnabled(true);
        if (inputEl) {
          inputEl.focus();
        }
      })
      .catch(function (err) {
        pending.remove();
        appendMessage("system", "System", err.message || "Something went wrong. Try again or use the enquiry form.");
        history.pop();
        setComposerEnabled(true);
        if (inputEl) {
          inputEl.focus();
        }
      });
  }

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

  connect();
})();
