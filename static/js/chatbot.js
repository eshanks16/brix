(function () {
    'use strict';

    // Conversation history sent with each request so Brix remembers context.
    var history = [];
    var isWaiting = false;

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    function init() {
        var fab = buildFAB();
        var win = buildWindow();
        document.body.appendChild(fab);
        document.body.appendChild(win);

        fab.addEventListener('click', toggleChat);
        win.querySelector('.chatbot-close').addEventListener('click', toggleChat);
        win.querySelector('.chatbot-send').addEventListener('click', handleSend);
        win.querySelector('.chatbot-input').addEventListener('keydown', function (e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSend();
            }
        });

        // Brix greets the user the first time the window is built.
        appendMessage('brix', "Hey there, pizza pal! 🍕 I'm Brix — your guide to everything delicious here at Brix Pizza. Ask me about our styles, toppings, or specialty pies and I'll point you in the right direction!");
    }

    // ── DOM builders ────────────────────────────────────────────────────────

    function buildFAB() {
        var btn = document.createElement('button');
        btn.id = 'chatbot-fab';
        btn.setAttribute('aria-label', 'Chat with Brix');
        btn.title = 'Chat with Brix!';
        btn.innerHTML =
            '<img src="/static/img/brix.png" alt="Brix mascot">' +
            '<span class="fab-label">Ask Brix!</span>';
        return btn;
    }

    function buildWindow() {
        var win = document.createElement('div');
        win.id = 'chatbot-window';
        win.setAttribute('role', 'dialog');
        win.setAttribute('aria-label', 'Chat with Brix');
        win.innerHTML =
            '<div class="chatbot-header">' +
            '  <div class="chatbot-header-info">' +
            '    <img src="/static/img/brix.png" alt="Brix" class="chatbot-header-avatar">' +
            '    <div>' +
            '      <div class="chatbot-header-name">Brix 🍕</div>' +
            '      <div class="chatbot-header-status">Your Pizza Pal</div>' +
            '    </div>' +
            '  </div>' +
            '  <button class="chatbot-close" aria-label="Close chat">&times;</button>' +
            '</div>' +
            '<div class="chatbot-messages" id="chatbot-messages" aria-live="polite"></div>' +
            '<div class="chatbot-input-row">' +
            '  <input class="chatbot-input" type="text" placeholder="Ask Brix anything…" maxlength="500" autocomplete="off">' +
            '  <button class="chatbot-send" aria-label="Send message">&#9658;</button>' +
            '</div>';
        return win;
    }

    // ── Open / close ─────────────────────────────────────────────────────────

    function toggleChat() {
        var win = document.getElementById('chatbot-window');
        var isOpen = win.classList.toggle('chatbot-open');
        if (isOpen) {
            document.querySelector('.chatbot-input').focus();
            scrollToBottom();
        }
    }

    // ── Send ─────────────────────────────────────────────────────────────────

    function handleSend() {
        if (isWaiting) return;

        var input = document.querySelector('.chatbot-input');
        var text = input.value.trim();
        if (!text) return;

        input.value = '';
        appendMessage('user', text);

        // Capture history before the new turn; the server appends the user
        // message itself so we only forward previous context.
        var contextHistory = history.slice();

        isWaiting = true;
        setInputEnabled(false);
        showTyping();

        fetch('/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message: text, history: contextHistory })
        })
        .then(function (res) {
            if (!res.ok) throw new Error('status ' + res.status);
            return res.json();
        })
        .then(function (data) {
            hideTyping();
            appendMessage('brix', data.response);
            // Record the completed exchange in history for future turns.
            history.push({ role: 'user', content: text });
            history.push({ role: 'assistant', content: data.response });
            // Keep at most 20 messages (10 exchanges) to bound context size.
            if (history.length > 20) {
                history = history.slice(history.length - 20);
            }
        })
        .catch(function () {
            hideTyping();
            appendMessage('brix', "Oops! Looks like I got distracted by a hot pizza fresh out of the oven. 🔥 Mind trying that again?");
        })
        .then(function () {
            isWaiting = false;
            setInputEnabled(true);
            document.querySelector('.chatbot-input').focus();
        });
    }

    // ── Message rendering ────────────────────────────────────────────────────

    function appendMessage(role, text) {
        var list = document.getElementById('chatbot-messages');
        var wrap = document.createElement('div');
        wrap.className = 'chatbot-msg ' + (role === 'brix' ? 'from-brix' : 'from-user');

        if (role === 'brix') {
            wrap.innerHTML =
                '<img src="/static/img/brix.png" alt="Brix" class="chatbot-msg-avatar">' +
                '<div class="chatbot-bubble">' + escapeHtml(text) + '</div>';
        } else {
            wrap.innerHTML =
                '<div class="chatbot-bubble">' + escapeHtml(text) + '</div>';
        }

        list.appendChild(wrap);
        scrollToBottom();
    }

    function showTyping() {
        var list = document.getElementById('chatbot-messages');
        var indicator = document.createElement('div');
        indicator.className = 'chatbot-msg from-brix';
        indicator.id = 'chatbot-typing';
        indicator.innerHTML =
            '<img src="/static/img/brix.png" alt="Brix" class="chatbot-msg-avatar">' +
            '<div class="chatbot-bubble"><div class="chatbot-typing">' +
            '<span></span><span></span><span></span>' +
            '</div></div>';
        list.appendChild(indicator);
        scrollToBottom();
    }

    function hideTyping() {
        var el = document.getElementById('chatbot-typing');
        if (el) el.remove();
    }

    // ── Helpers ──────────────────────────────────────────────────────────────

    function scrollToBottom() {
        var list = document.getElementById('chatbot-messages');
        if (list) list.scrollTop = list.scrollHeight;
    }

    function setInputEnabled(enabled) {
        var input = document.querySelector('.chatbot-input');
        var btn = document.querySelector('.chatbot-send');
        if (input) input.disabled = !enabled;
        if (btn) btn.disabled = !enabled;
    }

    function escapeHtml(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }
})();
