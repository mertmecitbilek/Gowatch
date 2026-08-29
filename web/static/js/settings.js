/**
 * GoWatch Settings — Notification channel management
 */

document.addEventListener('DOMContentLoaded', () => {
    const overlay = document.getElementById('modal-overlay');
    const addBtn = document.getElementById('add-notif-btn');
    const closeBtn = document.getElementById('modal-close');
    const cancelBtn = document.getElementById('cancel-btn');
    const form = document.getElementById('add-notif-form');
    const typeSelect = document.getElementById('n-type');

    // Modal handlers
    addBtn?.addEventListener('click', () => {
        overlay.classList.add('active');
        document.getElementById('n-name').focus();
    });

    function close() {
        overlay.classList.remove('active');
        form.reset();
        showTypeConfig('telegram');
    }

    closeBtn?.addEventListener('click', close);
    cancelBtn?.addEventListener('click', close);
    overlay?.addEventListener('click', e => { if (e.target === overlay) close(); });
    document.addEventListener('keydown', e => { if (e.key === 'Escape') close(); });

    // Tip değiştikçe config alanlarını göster
    typeSelect?.addEventListener('change', () => showTypeConfig(typeSelect.value));

    function showTypeConfig(type) {
        document.getElementById('config-telegram').style.display = '';
    }

    // Form submit
    form?.addEventListener('submit', async (e) => {
        e.preventDefault();

        const type = document.getElementById('n-type').value;
        let config = {};

        config = {
            bot_token: document.getElementById('tg-token').value.trim(),
            chat_id: document.getElementById('tg-chat').value.trim()
        };

        const payload = {
            name: document.getElementById('n-name').value.trim(),
            type,
            config: JSON.stringify(config),
        };

        try {
            const resp = await fetch('/api/notifications', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (resp.ok) {
                showToast('Notification channel saved!', 'success');
                close();
                setTimeout(() => location.reload(), 600);
            } else {
                const data = await resp.json();
                showToast(data.error || 'Failed to save', 'error');
            }
        } catch (err) {
            showToast('Network error', 'error');
        }
    });
});

// Bildirim kanalı sil
async function deleteNotification(id) {
    if (!confirm('Delete this notification channel?')) return;
    try {
        const resp = await fetch(`/api/notifications/${id}`, { method: 'DELETE' });
        if (resp.ok) {
            showToast('Notification channel deleted', 'success');
            setTimeout(() => location.reload(), 600);
        }
    } catch (err) {
        showToast('Error: ' + err.message, 'error');
    }
}
