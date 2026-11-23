const $ = (selector) => document.querySelector(selector);
const formatTime = (value) => new Date(value).toLocaleString();
async function request(path, options = {}) {
  const response = await fetch(path, { headers: { 'Content-Type': 'application/json' }, ...options });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error || `Request failed (${response.status})`);
  return body;
}
function showNotice(message, error = false) {
  const box = $('#notice'); box.textContent = message; box.hidden = false; box.classList.toggle('error', error);
  window.setTimeout(() => { box.hidden = true; }, 5000);
}
function escapeHTML(value) { return String(value).replace(/[&<>"']/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char])); }
async function refresh() {
  try {
    const [services, incidents, stats] = await Promise.all([
      request('/api/services'), request('/api/incidents'), request('/api/stats')
    ]);
    $('#stat-services').textContent = stats.services;
    $('#stat-healthy').textContent = stats.healthy;
    $('#stat-incidents').textContent = stats.open_incidents;
    $('#stat-recoveries').textContent = stats.recoveries;
    $('#nav-incidents').textContent = stats.open_incidents;
    $('#updated-at').textContent = `Updated ${new Date().toLocaleTimeString()}`;
    $('#services-body').innerHTML = services.length ? services.map(service => `
      <tr><td><span class="service-name"><span class="service-icon">${escapeHTML(service.icon)}</span>${escapeHTML(service.name)}</span></td>
      <td>${escapeHTML(service.environment)}</td><td><span class="badge ${service.status}">${escapeHTML(service.status)}</span></td>
      <td>${formatTime(service.last_check)}</td><td>
      ${service.status === 'healthy' ? `<button class="small-button warn" data-action="fail" data-name="${escapeHTML(service.name)}">Simulate failure</button>` : `<button class="small-button" data-action="recover" data-name="${escapeHTML(service.name)}">Recover</button>`}
      </td></tr>`).join('') : '<tr><td colspan="5" class="empty">No services registered</td></tr>';
    $('#incidents-list').innerHTML = incidents.length ? incidents.map(incident => `
      <article class="incident"><span class="incident-mark ${incident.status === 'resolved' ? 'success' : ''}">${incident.status === 'resolved' ? '✓' : '!'}</span>
      <div><div class="incident-title">${escapeHTML(incident.title)}</div><div class="incident-detail">${escapeHTML(incident.service)} · ${escapeHTML(incident.detail)}</div></div>
      <time class="incident-time">${formatTime(incident.created_at)}</time></article>`).join('') : '<div class="empty">No incidents. Services are operating normally.</div>';
  } catch (error) { showNotice(error.message, true); }
}
document.addEventListener('click', async (event) => {
  const button = event.target.closest('[data-action]');
  if (!button) return;
  button.disabled = true;
  try {
    const action = button.dataset.action;
    const result = await request(`/api/services/${encodeURIComponent(button.dataset.name)}/${action}`, { method: 'POST' });
    showNotice(result.message); await refresh();
  } catch (error) { showNotice(error.message, true); button.disabled = false; }
});
$('#refresh').addEventListener('click', refresh);
$('#refresh-main').addEventListener('click', refresh);
refresh(); window.setInterval(refresh, 15000);
