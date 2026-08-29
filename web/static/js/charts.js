/**
 * GoWatch Charts — Chart.js ile response time grafiği
 */

function initLatencyChart(heartbeats) {
    const ctx = document.getElementById('latencyChart');
    if (!ctx) return;

    const labels = heartbeats.map(hb => {
        const d = new Date(hb.time);
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    });

    const data = heartbeats.map(hb => hb.latency);
    const colors = heartbeats.map(hb => hb.status === 1 ? 'rgba(16,185,129,0.8)' : 'rgba(239,68,68,0.8)');

    new Chart(ctx, {
        type: 'bar',
        data: {
            labels,
            datasets: [{
                label: 'Response Time (ms)',
                data,
                backgroundColor: colors,
                borderRadius: 3,
                borderSkipped: false,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: 'rgba(22,27,34,0.95)',
                    borderColor: 'rgba(255,255,255,0.08)',
                    borderWidth: 1,
                    titleColor: '#e2e8f0',
                    bodyColor: '#8892a4',
                    callbacks: {
                        label: (ctx) => ` ${ctx.raw}ms`,
                    }
                }
            },
            scales: {
                x: {
                    ticks: {
                        color: '#4a5568',
                        maxTicksLimit: 12,
                        font: { size: 11 }
                    },
                    grid: { color: 'rgba(255,255,255,0.04)' },
                },
                y: {
                    ticks: {
                        color: '#4a5568',
                        font: { size: 11 },
                        callback: (val) => `${val}ms`
                    },
                    grid: { color: 'rgba(255,255,255,0.04)' },
                    beginAtZero: true,
                }
            }
        }
    });
}
