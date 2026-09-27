/**
 * GoWatch WebSocket Client
 * Gerçek zamanlı monitor durumu güncellemeleri
 */

class GoWatchWS {
    constructor() {
        this.ws = null;
        this.reconnectDelay = 2000;
        this.maxReconnectDelay = 30000;
        this.reconnectAttempts = 0;
        this.connect();
    }

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
            console.log('[GoWatch WS] Connected');
            this.reconnectAttempts = 0;
            this.reconnectDelay = 2000;
            this.updateConnectionStatus(true);
        };

        this.ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                this.handleMessage(msg);
            } catch (e) {
                console.error('[GoWatch WS] Parse error:', e);
            }
        };

        this.ws.onclose = () => {
            console.log('[GoWatch WS] Disconnected, reconnecting...');
            this.updateConnectionStatus(false);
            this.scheduleReconnect();
        };

        this.ws.onerror = (err) => {
            console.error('[GoWatch WS] Error:', err);
        };
    }

    handleMessage(msg) {
        if (msg.type === 'heartbeat' && msg.payload) {
            this.updateMonitor(msg.payload);
        }
    }

    updateMonitor(data) {
        const { monitor_id, status, latency, uptime_percent, status_changed, monitor_name } = data;

        // Monitor kartını güncelle (dashboard'daysa)
        const card = document.getElementById(`monitor-${monitor_id}`);
        if (card) {
            // Durum güncelle
            card.dataset.status = status;
            card.className = `monitor-card`;

            // Border rengi
            card.style.borderLeftColor = status === 1 ? 'var(--color-up)' :
                                          status === 0 ? 'var(--color-down)' : 'var(--color-pending)';

            // Pulse animasyon
            const pulse = card.querySelector('.status-pulse');
            if (pulse) {
                pulse.className = `status-pulse ${status === 1 ? 'pulse-up' : status === 0 ? 'pulse-down' : 'pulse-pending'}`;
            }

            // Badge güncelle
            const badge = document.getElementById(`badge-${monitor_id}`);
            if (badge) {
                const labels = { 1: 'UP', 0: 'DOWN', 2: 'PENDING' };
                const classes = { 1: 'badge-up', 0: 'badge-down', 2: 'badge-pending' };
                badge.textContent = labels[status] || 'UNKNOWN';
                badge.className = `monitor-status-badge ${classes[status] || 'badge-pending'}`;
            }

            // Latency güncelle
            const latencyEl = card.querySelector(`.latency-${monitor_id}`);
            if (latencyEl) latencyEl.textContent = `${latency}ms`;

            // Uptime güncelle
            const uptimeEl = card.querySelector(`.uptime-${monitor_id}`);
            if (uptimeEl) uptimeEl.textContent = `${uptime_percent ? uptime_percent.toFixed(2) : 0}%`;
        }

        // İstatistikleri güncelle
        this.updateStats();

        // Durum değişikliği bildirimi
        if (status_changed) {
            const statusText = status === 1 ? 'is back UP ✅' : 'is DOWN 🔴';
            showToast(`${monitor_name} ${statusText}`, status === 1 ? 'success' : 'error');
        }
    }

    updateStats() {
        const cards = document.querySelectorAll('.monitor-card');
        let up = 0, down = 0, pending = 0;

        cards.forEach(card => {
            const s = parseInt(card.dataset.status);
            if (s === 1) up++;
            else if (s === 0) down++;
            else pending++;
        });

        const el = (id, val) => {
            const e = document.getElementById(id);
            if (e) e.textContent = val;
        };

        el('stat-up', up);
        el('stat-down', down);
        el('stat-pending', pending);
        el('stat-total', cards.length);
    }

    updateConnectionStatus(connected) {
        const dot = document.getElementById('ws-dot');
        const label = document.getElementById('ws-label');
        if (dot) {
            dot.className = `status-dot ${connected ? 'connected' : 'disconnected'}`;
        }
        if (label) {
            label.textContent = connected ? 'Live' : 'Disconnected';
        }
    }

    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1), this.maxReconnectDelay);
        console.log(`[GoWatch WS] Reconnecting in ${delay}ms...`);
        setTimeout(() => this.connect(), delay);
    }
}

// Toast notification helper
function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const icons = {
        success: '✅',
        error: '🔴',
        info: 'ℹ️'
    };

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    // Mesaj kullanıcı girdisi (monitör adı) içerebilir; HTML olarak değil metin olarak eklenir
    const icon = document.createElement('span');
    icon.textContent = icons[type] || '';
    const text = document.createElement('span');
    text.textContent = message;
    toast.append(icon, text);
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.animation = 'none';
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(20px)';
        toast.style.transition = 'all 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

// Global WS instance
let gwWS;
document.addEventListener('DOMContentLoaded', () => {
    // Sadece authenticated sayfalarda WS başlat
    if (document.getElementById('ws-status') || document.getElementById('monitors-list')) {
        gwWS = new GoWatchWS();
    }
});
