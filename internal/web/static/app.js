const ENTITIES = [
  { id: 'vacancies',    label: 'Вакансии' },
  { id: 'resumes',      label: 'Резюме' },
  { id: 'applications', label: 'Отклики' },
  { id: 'seekers',      label: 'Соискатели' },
  { id: 'employers',    label: 'Работодатели' },
  { id: 'skills',       label: 'Навыки' },
  { id: 'reviews',      label: 'Отзывы' },
  { id: 'messages',     label: 'Сообщения' },
  { id: 'moderators',   label: 'Модераторы' },
];

const FIELD_LABELS = {
  id: 'ID', created_at: 'Создано', updated_at: 'Обновлено', status: 'Статус',
  email: 'Email', full_name: 'ФИО', phone: 'Телефон', city: 'Город',
  birth_date: 'Дата рождения', about: 'О себе',
  company_name: 'Компания', inn: 'ИНН', website: 'Сайт',
  verification_status: 'Верификация', moderator_id: 'ID модератора',
  seeker_id: 'ID соискателя', position: 'Должность',
  desired_salary: 'Желаемая зарплата', category: 'Категория',
  employment_type: 'Занятость', schedule: 'Формат',
  work_experience: 'Опыт работы', education: 'Образование',
  employer_id: 'ID работодателя', title: 'Название', description: 'Описание',
  salary_from: 'Зарплата от', salary_to: 'Зарплата до',
  required_experience: 'Требуемый опыт',
  name: 'Название', resume_id: 'ID резюме', vacancy_id: 'ID вакансии',
  skill_id: 'ID навыка', cover_letter: 'Сопроводительное письмо',
  initiator: 'Инициатор',
  author_type: 'Тип автора', author_id: 'ID автора', text: 'Текст',
  is_read: 'Прочитано', rating: 'Рейтинг', is_published: 'Опубликован',
};

const VALUE_LABELS = {
  status: { active:'активен', pending:'на модерации', rejected:'отклонён', archived:'в архиве', draft:'черновик', viewed:'просмотрен', invited:'приглашён', hired:'принят', blocked:'заблокирован', deleted:'удалён' },
  verification_status: { pending:'на проверке', approved:'одобрен', rejected:'отклонён' },
  employment_type: { full:'полная', part:'частичная', contract:'контракт', internship:'стажировка' },
  schedule: { remote:'удалённо', office:'офис', hybrid:'гибрид' },
  author_type: { seeker:'соискатель', employer:'работодатель' },
  initiator: { seeker:'соискатель', employer:'работодатель' },
};

const ENUM_FIELDS = {
  status: ['active','pending','rejected','archived','draft','viewed','invited','hired','blocked','deleted'],
  verification_status: ['pending','approved','rejected'],
  employment_type: ['full','part','contract','internship'],
  schedule: ['remote','office','hybrid'],
  author_type: ['seeker','employer'],
  initiator: ['seeker','employer'],
};

const SEARCH_FIELDS = {
  vacancies: [
    { name:'id', label:'ID', type:'int' },
    { name:'title', label:'Название', type:'str' },
    { name:'city', label:'Город', type:'str' },
    { name:'status', label:'Статус', type:'select', options:['active','pending','rejected','archived'] },
    { name:'employment_type', label:'Занятость', type:'select', options:['full','part','contract','internship'] },
    { name:'schedule', label:'Формат', type:'select', options:['remote','office','hybrid'] },
    { name:'employer_id', label:'ID работодателя', type:'int' },
    { name:'salary_from', label:'Зарплата от', type:'int' },
  ],
  resumes: [
    { name:'id', label:'ID', type:'int' },
    { name:'position', label:'Должность', type:'str' },
    { name:'status', label:'Статус', type:'select', options:['active','archived','draft'] },
    { name:'seeker_id', label:'ID соискателя', type:'int' },
    { name:'category', label:'Категория', type:'str' },
    { name:'desired_salary', label:'Желаемая зарплата', type:'int' },
  ],
  applications: [
    { name:'id', label:'ID', type:'int' },
    { name:'vacancy_id', label:'ID вакансии', type:'int' },
    { name:'seeker_id', label:'ID соискателя', type:'int' },
    { name:'status', label:'Статус', type:'select', options:['pending','viewed','invited','rejected','hired'] },
    { name:'created_at', label:'Создано (unix)', type:'int' },
  ],
  seekers: [
    { name:'id', label:'ID', type:'int' },
    { name:'email', label:'Email', type:'str' },
    { name:'city', label:'Город', type:'str' },
    { name:'status', label:'Статус', type:'select', options:['active','blocked','deleted'] },
  ],
  employers: [
    { name:'id', label:'ID', type:'int' },
    { name:'email', label:'Email', type:'str' },
    { name:'company_name', label:'Компания', type:'str' },
    { name:'city', label:'Город', type:'str' },
    { name:'verification_status', label:'Верификация', type:'select', options:['pending','approved','rejected'] },
  ],
  skills: [
    { name:'id', label:'ID', type:'int' },
    { name:'name', label:'Название', type:'str' },
  ],
  reviews: [
    { name:'id', label:'ID', type:'int' },
    { name:'employer_id', label:'ID работодателя', type:'int' },
    { name:'seeker_id', label:'ID соискателя', type:'int' },
    { name:'rating', label:'Рейтинг', type:'int' },
    { name:'is_published', label:'Опубликован', type:'bool' },
  ],
  messages: [
    { name:'id', label:'ID', type:'int' },
    { name:'author_type', label:'Тип автора', type:'select', options:['seeker','employer'] },
    { name:'author_id', label:'ID автора', type:'int' },
    { name:'is_read', label:'Прочитано', type:'bool' },
    { name:'sent_at', label:'Отправлено (unix)', type:'int' },
  ],
  moderators: [
    { name:'id', label:'ID', type:'int' },
    { name:'email', label:'Email', type:'str' },
    { name:'full_name', label:'ФИО', type:'str' },
  ],
};

const state = {
  entity: 'vacancies',
  fields: [],
  rows: [],
  selected: null,
  dialogEditing: false,
  skillsMap: {},
  allSkills: null,
};

let formSkills = [];
let formSkillsInitial = [];

function fieldLabel(n) { return FIELD_LABELS[n] || n; }
function valueLabel(f, v) { const m = VALUE_LABELS[f]; return (m && typeof v === 'string' && m[v]) ? m[v] : v; }
function entityLabel(id) { return ENTITIES.find(e => e.id === id)?.label || id; }
function hasSkillsEntity() { return state.entity === 'resumes' || state.entity === 'vacancies'; }

function sortByPK(rows) {
  if (!Array.isArray(rows)) return [];
  return [...rows].sort((a, b) => {
    const ai = Number(a?.id ?? 0), bi = Number(b?.id ?? 0);
    if (Number.isFinite(ai) && Number.isFinite(bi)) return ai - bi;
    return String(a?.id ?? '').localeCompare(String(b?.id ?? ''));
  });
}

function fieldIndex(name) {
  const f = state.fields.find(x => x.name === name);
  return f?.index || '';
}

async function ensureSkillsLoaded(force = false) {
  if (state.allSkills && !force) return state.allSkills;
  const r = await fetch('/api/skills');
  const j = await r.json();
  state.allSkills = j.rows || [];
  return state.allSkills;
}
function ensureDatalist() {
  let dl = document.getElementById('skills-datalist');
  if (!dl) {
    dl = document.createElement('datalist');
    dl.id = 'skills-datalist';
    document.body.appendChild(dl);
  }
  return dl;
}

function renderTable() {
  const thead = document.querySelector('#table thead');
  const tbody = document.querySelector('#table tbody');
  thead.innerHTML = ''; tbody.innerHTML = '';
  document.getElementById('empty').classList.toggle('hidden', state.rows.length > 0);
  if (state.rows.length === 0) return;

  const hasSkills = hasSkillsEntity();
  const cols = state.fields.map(f => f.name);
  if (hasSkills) cols.push('__skills__');

  const trh = document.createElement('tr');
  for (const c of cols) {
    const th = document.createElement('th');
    th.textContent = c === '__skills__' ? 'Навыки' : fieldLabel(c);
    trh.appendChild(th);
  }
  thead.appendChild(trh);

  for (const row of state.rows) {
    const tr = document.createElement('tr');
    tr.onclick = () => { state.selected = row; highlight(tr); };
    if (state.selected && state.selected.id === row.id) tr.classList.add('selected');
    for (const f of state.fields) {
      const td = document.createElement('td');
      td.textContent = formatCell(row[f.name], f.type, f.name);
      td.title = td.textContent;
      tr.appendChild(td);
    }
    if (hasSkills) {
      const td = document.createElement('td');
      const arr = state.skillsMap[row.id] || [];
      td.textContent = arr.length ? arr.map(s => s.name).join(', ') : '-';
      td.title = td.textContent;
      if (arr.length === 0) td.style.color = '#94a3b8';
      tr.appendChild(td);
    }
    tbody.appendChild(tr);
  }
}

async function loadStats() {
  try {
    const r = await fetch('/api/counts');
    const j = await r.json();
    const parts = ENTITIES.map(e => `${e.label.toLowerCase()}: ${j[e.id] || 0}`);
    document.getElementById('stats').textContent = ' • ' + parts.join(' • ');
  } catch {}
}

async function load() {
  setStatus('загрузка…');
  const r = await fetch(`/api/${state.entity}`);
  const j = await r.json();
  if (j.error) { setStatus('ошибка: ' + j.error); return; }
  state.fields = j.fields || [];
  state.rows = sortByPK(j.rows || []);
  state.skillsMap = {};
  if (hasSkillsEntity()) await loadSkillsMap();
  renderTable();
  renderSearchPanel();
  loadStats();
  if (state.selected) {
    const id = state.selected.id;
    state.selected = state.rows.find(r => r.id === id) || null;
  }
  setStatus(`${state.rows.length} строк`);
}

async function loadSkillsMap() {
  const linkEntity = state.entity === 'resumes' ? 'resume_skills' : 'vacancy_skills';
  const parentField = state.entity === 'resumes' ? 'resume_id' : 'vacancy_id';
  const [linksResp, skillsResp] = await Promise.all([fetch(`/api/${linkEntity}`), fetch('/api/skills')]);
  const links = (await linksResp.json()).rows || [];
  const skills = (await skillsResp.json()).rows || [];
  state.allSkills = skills;
  const nameById = {};
  for (const s of skills) nameById[s.id] = s.name;
  for (const l of links) {
    const pid = l[parentField];
    if (!state.skillsMap[pid]) state.skillsMap[pid] = [];
    state.skillsMap[pid].push({ id: l.skill_id, name: nameById[l.skill_id] || `#${l.skill_id}` });
  }
}

function formatCell(v, type, fieldName) {
  if (v === null || v === undefined || v === '') return '';
  if (type === 'bool') return v ? 'да' : 'нет';
  if (type === 'date') { const n = Number(v); return (!n || isNaN(n)) ? '' : new Date(n * 1000).toISOString().slice(0, 10); }
  if (fieldName && typeof v === 'string') {
    const m = VALUE_LABELS[fieldName];
    if (m && m[v]) return m[v];
  }
  return String(v);
}

function highlight(tr) {
  document.querySelectorAll('#table tbody tr').forEach(r => r.classList.remove('selected'));
  tr.classList.add('selected');
}

function renderSearchPanel() {
  const panel = document.getElementById('search-panel');
  const cfg = SEARCH_FIELDS[state.entity] || [];
  if (cfg.length === 0) { panel.classList.add('hidden'); return; }
  panel.classList.remove('hidden');
  panel.innerHTML = '';

  const fieldWrap = document.createElement('label');
  const fieldSpan = document.createElement('span');
  fieldSpan.textContent = 'Поле поиска';
  const fieldSelect = document.createElement('select');
  fieldSelect.id = 'search-field';
  fieldSelect.innerHTML = cfg.map(f => `<option value="${f.name}">${f.label}</option>`).join('');
  fieldWrap.appendChild(fieldSpan);
  fieldWrap.appendChild(fieldSelect);
  panel.appendChild(fieldWrap);

  const valueWrap = document.createElement('label');
  const valueSpan = document.createElement('span');
  valueSpan.textContent = 'Значение';
  const valueBox = document.createElement('span');
  valueBox.id = 'search-value-box';
  valueWrap.appendChild(valueSpan);
  valueWrap.appendChild(valueBox);
  panel.appendChild(valueWrap);

  const rangeWrap = document.createElement('label');
  rangeWrap.style.display = 'none';
  rangeWrap.id = 'range-wrap';
  const rangeSpan = document.createElement('span');
  rangeSpan.textContent = 'Диапазон от';
  const rangeBox = document.createElement('span');
  rangeBox.innerHTML = `от <input id="range-lo" type="number" style="width:110px"> до <input id="range-hi" type="number" style="width:110px">`;
  rangeWrap.appendChild(rangeSpan);
  rangeWrap.appendChild(rangeBox);
  panel.appendChild(rangeWrap);

  function rebuildInput() {
    const f = cfg.find(x => x.name === fieldSelect.value);
    let el;
    if (f.type === 'select' || f.type === 'bool') {
      el = document.createElement('select');
      const opts = f.type === 'bool' ? ['true', 'false'] : f.options;
      el.innerHTML = '<option value="">-</option>' + opts.map(o => `<option value="${o}">${f.type === 'bool' ? (o === 'true' ? 'да' : 'нет') : valueLabel(f.name, o)}</option>`).join('');
    } else {
      el = document.createElement('input');
      el.type = f.type === 'int' ? 'number' : 'text';
      el.placeholder = 'значение';
    }
    el.id = 'search-value';
    valueBox.innerHTML = '';
    valueBox.appendChild(el);
    const isBTreeInt = (fieldIndex(f.name) === 'btree') && (f.type === 'int');
    rangeWrap.style.display = isBTreeInt ? '' : 'none';
  }
  fieldSelect.onchange = rebuildInput;
  rebuildInput();

  const btnSearch = document.createElement('button');
  btnSearch.textContent = 'Найти';
  btnSearch.className = 'primary';
  btnSearch.onclick = doServerSearch;
  panel.appendChild(btnSearch);

  const btnRange = document.createElement('button');
  btnRange.textContent = 'Найти диапазон';
  btnRange.className = 'primary';
  btnRange.onclick = doServerRange;
  panel.appendChild(btnRange);

  const btnDelete = document.createElement('button');
  btnDelete.textContent = 'Удалить найденные';
  btnDelete.className = 'danger';
  btnDelete.onclick = doServerDeleteByField;
  panel.appendChild(btnDelete);

  const btnReset = document.createElement('button');
  btnReset.textContent = 'Сбросить';
  btnReset.onclick = () => load();
  panel.appendChild(btnReset);
}

async function doServerSearch() {
  const field = document.getElementById('search-field').value;
  const valueEl = document.getElementById('search-value');
  const value = valueEl ? valueEl.value : '';
  if (value === '') { alert('Введите значение'); return; }
  setStatus('поиск…');
  const url = `/api/${state.entity}/search?field=${encodeURIComponent(field)}&value=${encodeURIComponent(value)}`;
  try {
    const r = await fetch(url);
    const j = await r.json();
    if (j.error) { alert('Ошибка: ' + j.error); setStatus(''); return; }
    state.rows = sortByPK(j.rows || []);
    state.selected = null;
    renderTable();
    setStatus(`найдено: ${state.rows.length}`);
  } catch (e) { alert('Сеть: ' + e.message); setStatus(''); }
}

async function doServerRange() {
  const field = document.getElementById('search-field').value;
  const lo = document.getElementById('range-lo')?.value ?? '';
  const hi = document.getElementById('range-hi')?.value ?? '';
  if (lo === '' && hi === '') { alert('Укажите хотя бы одну границу'); return; }
  const params = new URLSearchParams({ field });
  if (lo !== '') params.set('lo', lo);
  if (hi !== '') params.set('hi', hi);
  setStatus('диапазон…');
  try {
    const r = await fetch(`/api/${state.entity}/range?${params}`);
    const j = await r.json();
    if (j.error) { alert('Ошибка: ' + j.error); setStatus(''); return; }
    state.rows = sortByPK(j.rows || []);
    state.selected = null;
    renderTable();
    setStatus(`в диапазоне: ${state.rows.length}`);
  } catch (e) { alert('Сеть: ' + e.message); setStatus(''); }
}

async function doServerDeleteByField() {
  const field = document.getElementById('search-field').value;
  const valueEl = document.getElementById('search-value');
  const value = valueEl ? valueEl.value : '';
  if (value === '') { alert('Введите значение'); return; }
  if (!confirm(`Удалить все записи, где ${field} = «${value}»?`)) return;
  const url = `/api/${state.entity}/by-field?field=${encodeURIComponent(field)}&value=${encodeURIComponent(value)}`;
  try {
    const r = await fetch(url, { method: 'DELETE' });
    const j = await r.json();
    if (j.error) { alert('Ошибка: ' + j.error); return; }
    let msg = `Удалено: ${j.deleted}`;
    if (j.errors && j.errors.length) msg += `\n\nОшибки:\n${j.errors.join('\n')}`;
    alert(msg);
    load();
  } catch (e) { alert('Сеть: ' + e.message); }
}

function renderTabs() {
  const nav = document.getElementById('tabs');
  nav.innerHTML = '';
  for (const e of ENTITIES) {
    const b = document.createElement('button');
    b.textContent = e.label;
    if (e.id === state.entity) b.classList.add('active');
    b.onclick = () => {
      if (state.entity === 'skills') state.allSkills = null;
      state.entity = e.id;
      state.selected = null;
      load();
      renderTabs();
    };
    nav.appendChild(b);
  }
}

document.getElementById('btn-add').onclick = () => openForm(null);
document.getElementById('btn-edit').onclick = () => {
  if (!state.selected) { setStatus('выберите строку'); return; }
  openForm(state.selected);
};
document.getElementById('btn-del').onclick = async () => {
  if (!state.selected) { setStatus('выберите строку'); return; }
  const pk = state.selected.id;
  if (!confirm(`Удалить ${entityLabel(state.entity)} id=${pk}?`)) return;
  const r = await fetch(`/api/${state.entity}/${pk}`, { method: 'DELETE' });
  const j = await r.json();
  if (j.error) { alert(j.error); return; }
  state.selected = null;
  load();
};
document.getElementById('btn-save').onclick = async () => {
  const r = await fetch('/api/db/save', { method: 'POST' });
  const j = await r.json();
  if (j.error) { alert('Ошибка: ' + j.error); return; }
  setStatus('сохранено в ' + new Date().toLocaleTimeString());
};
document.getElementById('btn-reload').onclick = () => load();
document.getElementById('btn-xlsx').onclick = () => { window.location.href = `/api/export/xlsx?entity=${state.entity}`; };
document.getElementById('btn-backup').onclick = () => { window.location.href = '/api/backup'; };
document.getElementById('in-restore').onchange = async (e) => {
  const file = e.target.files[0];
  if (!file) return;
  if (!confirm('Восстановить из ' + file.name + '?')) return;
  const buf = await file.arrayBuffer();
  const r = await fetch('/api/restore', { method: 'POST', body: buf });
  const j = await r.json();
  if (j.error) { alert(j.error); return; }
  alert('Восстановлено. Перезапустите приложение.');
  e.target.value = '';
};
document.getElementById('btn-clear-db').onclick = async () => {
  if (!confirm('Очистить ВСЕ базы данных? Все записи будут удалены, схемы останутся.')) return;
  const r = await fetch('/api/db/clear', { method: 'POST' });
  const j = await r.json();
  if (j.error) { alert('Ошибка: ' + j.error); return; }
  state.rows = []; state.selected = null;
  renderTable(); loadStats();
  setStatus('все БД очищены');
};
document.getElementById('btn-delete-db').onclick = async () => {
  if (!confirm('Удалить ВСЕ файлы БД? Данные потеряны безвозвратно.')) return;
  if (!confirm('Точно уверены?')) return;
  const r = await fetch('/api/db/delete', { method: 'POST' });
  const j = await r.json();
  if (j.error) { alert('Ошибка: ' + j.error); return; }
  alert(j.message || 'Все БД удалены. Перезапустите приложение.');
  state.rows = []; state.selected = null;
  renderTable();
  setStatus('файлы БД удалены - перезапустите приложение');
};

const dlg = document.getElementById('dialog');
const form = document.getElementById('form');
const dlgTitle = document.getElementById('dlg-title');

async function openForm(row) {
  state.dialogEditing = !!row;
  dlgTitle.textContent = (row ? 'Изменить' : 'Добавить') + ' - ' + entityLabel(state.entity);
  form.innerHTML = '';
  formSkills = []; formSkillsInitial = [];
  for (const f of state.fields) {
    if (f.name === 'id' && !row) continue;
    const label = document.createElement('label');
    const span = document.createElement('span');
    span.textContent = fieldLabel(f.name);
    label.appendChild(span);
    const val = row ? row[f.name] : undefined;
    let input;
    const enumOpts = ENUM_FIELDS[f.name];
    if (enumOpts) {
      input = document.createElement('select');
      const opts = enumOpts.slice();
      if (val !== undefined && val !== null && val !== '' && !opts.includes(String(val))) opts.push(String(val));
      input.innerHTML = '<option value="">-</option>' + opts.map(o => `<option value="${o}">${valueLabel(f.name, o)}</option>`).join('');
      if (val !== undefined && val !== null && val !== '') input.value = String(val);
    } else {
      switch (f.type) {
        case 'bool':
          input = document.createElement('select');
          input.innerHTML = '<option value="true">да</option><option value="false">нет</option>';
          input.value = String(val === true);
          break;
        case 'int64':
        case 'float64':
          input = document.createElement('input');
          input.type = 'number';
          input.value = (val !== null && val !== undefined) ? val : '';
          break;
        case 'date':
          input = document.createElement('input');
          input.type = 'date';
          if (val && Number(val) > 0) input.value = new Date(Number(val) * 1000).toISOString().slice(0, 10);
          break;
        default: {
          const isLong = ['description','text','cover_letter','work_experience','education','required_experience','about'].includes(f.name);
          input = document.createElement(isLong ? 'textarea' : 'input');
          if (isLong) input.rows = 4; else input.type = 'text';
          input.value = (val !== null && val !== undefined) ? String(val) : '';
        }
      }
    }
    input.dataset.field = f.name;
    input.dataset.type = f.type;
    if (f.name === 'id' && row) input.disabled = true;
    label.appendChild(input);
    form.appendChild(label);
  }
  if (hasSkillsEntity()) await renderFormSkills(row);
  dlg.showModal();
}

async function renderFormSkills(row) {
  await ensureSkillsLoaded();
  const dl = ensureDatalist();
  if (row) {
    const existing = state.skillsMap[row.id] || [];
    formSkillsInitial = existing.map(s => ({ id: s.id, name: s.name }));
    formSkills = existing.map(s => ({ id: s.id, name: s.name }));
  }
  const section = document.createElement('div');
  section.className = 'form-skills-section';
  const title = document.createElement('span');
  title.className = 'form-skills-title';
  title.textContent = 'Навыки';
  section.appendChild(title);
  const chipsBox = document.createElement('div');
  chipsBox.className = 'form-skills-chips';
  const inputRow = document.createElement('div');
  inputRow.className = 'form-skills-input-row';
  const input = document.createElement('input');
  input.type = 'text';
  input.placeholder = 'начните вводить название навыка…';
  input.setAttribute('list', 'skills-datalist');
  input.autocomplete = 'off';

  function redraw() {
    chipsBox.innerHTML = '';
    if (formSkills.length === 0) {
      const em = document.createElement('span');
      em.className = 'form-skills-empty';
      em.textContent = 'навыки не выбраны';
      chipsBox.appendChild(em);
    } else {
      for (const s of formSkills) {
        const chip = document.createElement('span');
        chip.className = 'form-skill-chip';
        const txt = document.createElement('span');
        txt.textContent = s.name;
        chip.appendChild(txt);
        const btn = document.createElement('button');
        btn.type = 'button'; btn.textContent = '×'; btn.title = 'убрать';
        btn.onclick = (e) => { e.preventDefault(); formSkills = formSkills.filter(x => x.id !== s.id); redraw(); rebuildDatalist(); };
        chip.appendChild(btn);
        chipsBox.appendChild(chip);
      }
    }
  }
  function rebuildDatalist() {
    dl.innerHTML = '';
    const taken = new Set(formSkills.map(s => s.id));
    for (const s of state.allSkills) if (!taken.has(s.id)) {
      const o = document.createElement('option');
      o.value = s.name;
      dl.appendChild(o);
    }
  }
  function tryAdd(raw) {
    const q = String(raw || '').trim();
    if (!q) return false;
    const found = state.allSkills.find(s => s.name.toLowerCase() === q.toLowerCase());
    if (!found || formSkills.some(x => x.id === found.id)) return false;
    formSkills.push({ id: found.id, name: found.name });
    redraw(); rebuildDatalist();
    return true;
  }
  input.addEventListener('input', () => {
    const v = input.value.trim();
    if (!v) return;
    const found = state.allSkills.find(s => s.name.toLowerCase() === v.toLowerCase());
    if (found && !formSkills.some(x => x.id === found.id)) { tryAdd(v); input.value = ''; }
  });
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ',' || e.key === ';') { e.preventDefault(); if (tryAdd(input.value)) input.value = ''; }
  });
  inputRow.appendChild(input);
  section.appendChild(chipsBox);
  section.appendChild(inputRow);
  form.appendChild(section);
  redraw(); rebuildDatalist();
}

async function applySkillDiff(recordId) {
  if (!hasSkillsEntity()) return [];
  const numId = Number(recordId);
  if (!Number.isInteger(numId) || numId <= 0) return ['не удалось определить ID записи'];
  const endpoint = state.entity === 'resumes' ? 'resumes-skills' : 'vacancies-skills';
  const parentField = state.entity === 'resumes' ? 'resume_id' : 'vacancy_id';
  const initIds = new Set(formSkillsInitial.map(s => Number(s.id)));
  const finalIds = new Set(formSkills.map(s => Number(s.id)));
  const toAdd = formSkills.filter(s => !initIds.has(Number(s.id)));
  const toRemove = formSkillsInitial.filter(s => !finalIds.has(Number(s.id)));
  const errors = [];
  for (const s of toRemove) {
    const sid = Number(s.id);
    const r = await fetch(`/api/${endpoint}/${numId}/${sid}`, {
      method: 'DELETE', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ [parentField]: numId, skill_id: sid }),
    });
    const j = await r.json();
    if (j.error && !/pair not found/i.test(j.error)) errors.push(`убрать «${s.name}»: ${j.error}`);
  }
  for (const s of toAdd) {
    const sid = Number(s.id);
    const r = await fetch(`/api/${endpoint}/${numId}/${sid}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ [parentField]: numId, skill_id: sid }),
    });
    const j = await r.json();
    if (j.error) errors.push(`добавить «${s.name}»: ${j.error}`);
  }
  return errors;
}

document.getElementById('dlg-cancel').onclick = () => dlg.close();
document.getElementById('dlg-ok').onclick = async () => {
  const payload = {};
  form.querySelectorAll('[data-field]').forEach(el => {
    const name = el.dataset.field, type = el.dataset.type;
    let v = el.value;
    switch (type) {
      case 'bool': payload[name] = v === 'true'; break;
      case 'int64': payload[name] = v === '' ? 0 : parseInt(v, 10); break;
      case 'float64': payload[name] = v === '' ? 0 : parseFloat(v); break;
      case 'date':
        if (v === '') { payload[name] = 0; break; }
        payload[name] = Math.floor(new Date(v + 'T00:00:00Z').getTime() / 1000);
        break;
      default:
        if (state.dialogEditing || v !== '') payload[name] = v;
    }
  });
  const editing = state.dialogEditing;
  const pk = editing ? state.selected.id : null;
  const url = editing ? `/api/${state.entity}/${pk}` : `/api/${state.entity}`;
  const method = editing ? 'PUT' : 'POST';
  const r = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
  const j = await r.json();
  if (j.error) { alert(j.error); return; }
  let savedId;
  if (editing) savedId = pk;
  else if (j.id !== undefined && j.id !== null) savedId = j.id;
  else {
    const r2 = await fetch(`/api/${state.entity}`);
    const j2 = await r2.json();
    const ids = (j2.rows || []).map(row => Number(row.id)).filter(n => !isNaN(n));
    if (ids.length) savedId = Math.max(...ids);
  }
  if (hasSkillsEntity()) {
    const errs = await applySkillDiff(savedId);
    if (errs && errs.length) alert('Запись сохранена, но были ошибки по навыкам:\n\n' + errs.join('\n'));
  }
  if (state.entity === 'skills') state.allSkills = null;
  dlg.close();
  load();
};

function setStatus(s) { document.getElementById('status').textContent = s; }

renderTabs();
load();