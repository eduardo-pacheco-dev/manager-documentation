import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Add, Calendar, Edit, OverflowMenuVertical, TrashCan } from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  InlineLoading,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  Search,
  Select,
  SelectItem,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage } from '../../lib/format';

interface AgendaShare { id: number; email: string; name: string; role: string }
interface AgendaCalendar {
  id: number; name: string; color: string; timezone: string; is_default: boolean; is_visible: boolean;
  role: string; event_count: number; shares: AgendaShare[];
}
interface Occurrence {
  id: number; recurrence_id: string | null; title: string; description: string | null; location: string | null;
  start: string; end: string; is_all_day: boolean; is_recurring: boolean; status: string; color: string;
  calendar_id: number; calendar_name: string; organizer_name: string | null;
}
interface AgendaData { calendars: AgendaCalendar[]; occurrences: Occurrence[]; colors: Record<string, string> }
interface EventDetail {
  id: number; calendar_id: number; title: string; description: string | null; location: string | null;
  meeting_url: string | null; start_at: string; end_at: string; is_all_day: boolean; status: string;
  visibility: string; transparency: string; color: string | null; recurrence: { frequency: string; interval: number; count?: number; until?: string } | null;
  recurrence_until: string | null; attendees: { email: string; name: string | null; status: string }[];
  can_modify?: boolean;
}

const WEEKDAYS = ['Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb', 'Dom'];
const STATUS_LABELS: Record<string, string> = { confirmed: 'Confirmado', tentative: 'Provisório', cancelled: 'Cancelado' };

function ymd(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function localInput(value: string): string {
  const date = new Date(value);
  return `${ymd(date)}T${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

function monthDays(reference: Date): Date[] {
  const first = new Date(reference.getFullYear(), reference.getMonth(), 1);
  const offset = (first.getDay() + 6) % 7;
  const start = new Date(first.getFullYear(), first.getMonth(), first.getDate() - offset);
  return Array.from({ length: 42 }, (_, index) => new Date(start.getFullYear(), start.getMonth(), start.getDate() + index));
}

function colorHex(name: string, colors: Record<string, string>): string {
  return colors[name] ?? colors.blueberry ?? '#3f51b5';
}

export function AgendaPage() {
  const toast = useToast();
  const [searchParams] = useSearchParams();
  const [calendars, setCalendars] = useState<AgendaCalendar[]>([]);
  const [occurrences, setOccurrences] = useState<Occurrence[]>([]);
  const [colors, setColors] = useState<Record<string, string>>({});
  const [date, setDate] = useState(new Date());
  const [view, setView] = useState<'month' | 'week' | 'day' | 'list'>('month');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [visible, setVisible] = useState<number[]>([]);
  const [eventModal, setEventModal] = useState(false);
  const [editing, setEditing] = useState<EventDetail | null>(null);
  const [calendarModal, setCalendarModal] = useState(false);
  const [calendarEditing, setCalendarEditing] = useState<AgendaCalendar | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    calendar_id: 0, title: '', description: '', location: '', meeting_url: '',
    start_at: '', end_at: '', is_all_day: false, status: 'confirmed', color: '',
    frequency: '', interval: 1, until: '', attendees: '', scope: 'all', recurrence_id: '',
  });
  const [calendarForm, setCalendarForm] = useState({ name: '', description: '', color: 'blueberry', timezone: 'America/Sao_Paulo', email: '', role: 'reader' });
  const [importCalendar, setImportCalendar] = useState(0);
  const [importModal, setImportModal] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);

  async function load() {
    setLoading(true);
    try {
      const response = await api.get<{ data: AgendaData }>(`/agenda?date=${ymd(date)}&view=${view}&search=${encodeURIComponent(search)}`);
      setCalendars(response.data.calendars);
      setOccurrences(response.data.occurrences);
      setColors(response.data.colors);
      if (visible.length === 0) setVisible(response.data.calendars.filter((calendar) => calendar.role !== 'owner' || calendar.is_visible).map((calendar) => calendar.id));
    } catch (error) {
      toast.error('Não foi possível carregar a agenda', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [date, view, search]);

  useEffect(() => {
    const id = Number(searchParams.get('event'));
    if (Number.isFinite(id) && id > 0) {
      void openEdit({ id, recurrence_id: null } as Occurrence);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const days = useMemo(() => monthDays(date), [date]);
  const filtered = occurrences.filter((occurrence) => visible.includes(occurrence.calendar_id));
  const byDay = useMemo(() => {
    const map: Record<string, Occurrence[]> = {};
    filtered.forEach((occurrence) => {
      const key = ymd(new Date(occurrence.start));
      map[key] = [...(map[key] ?? []), occurrence];
    });
    return map;
  }, [filtered]);

  function openCreate(day?: Date) {
    const start = day ? new Date(day.getFullYear(), day.getMonth(), day.getDate(), 9, 0) : new Date();
    const end = new Date(start.getTime() + 60 * 60 * 1000);
    setEditing(null);
    setForm({
      calendar_id: calendars.find((calendar) => calendar.role !== 'reader' && calendar.role !== 'free_busy_reader')?.id ?? calendars[0]?.id ?? 0,
      title: '', description: '', location: '', meeting_url: '', start_at: localInput(start.toISOString()),
      end_at: localInput(end.toISOString()), is_all_day: false, status: 'confirmed', color: '',
      frequency: '', interval: 1, until: '', attendees: '', scope: 'all', recurrence_id: '',
    });
    setEventModal(true);
  }

  function openCalendarCreate() {
    setCalendarEditing(null);
    setCalendarForm({ name: '', description: '', color: 'blueberry', timezone: 'America/Sao_Paulo', email: '', role: 'reader' });
    setCalendarModal(true);
  }

  function openImport() {
    setImportCalendar(calendars.find((calendar) => calendar.role !== 'reader' && calendar.role !== 'free_busy_reader')?.id ?? calendars[0]?.id ?? 0);
    setImportFile(null);
    setImportModal(true);
  }

  async function importICS() {
    if (!importFile) {
      toast.error('Selecione um arquivo .ics para importar');
      return;
    }
    setImporting(true);
    try {
      const data = new FormData();
      data.append('file', importFile);
      data.append('calendar_id', String(importCalendar || calendars[0]?.id || 0));
      const response = await api.upload<{ message: string }>('/agenda/ics/import', data);
      toast.success(response.message);
      setImportModal(false);
      setImportFile(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível importar', errorMessage(error));
    } finally {
      setImporting(false);
    }
  }

  async function openEdit(occurrence: Occurrence) {
    try {
      const response = await api.get<{ data: { event: EventDetail } }>(`/agenda/events/${occurrence.id}`);
      const event = response.data.event;
      setEditing(event);
      setForm({
        calendar_id: event.calendar_id, title: event.title, description: event.description ?? '',
        location: event.location ?? '', meeting_url: event.meeting_url ?? '',
        start_at: localInput(event.start_at), end_at: localInput(event.end_at), is_all_day: event.is_all_day,
        status: event.status, color: event.color ?? '', frequency: event.recurrence?.frequency ?? '',
        interval: event.recurrence?.interval ?? 1, until: event.recurrence_until ?? '',
        attendees: event.attendees.filter((attendee) => attendee.email).map((attendee) => attendee.email).join(', '),
        scope: 'all', recurrence_id: occurrence.recurrence_id ?? '',
      });
      setEventModal(true);
    } catch (error) {
      toast.error('Não foi possível abrir o evento', errorMessage(error));
    }
  }

  async function saveEvent() {
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        calendar_id: form.calendar_id, title: form.title, description: form.description,
        location: form.location, meeting_url: form.meeting_url, start_at: form.start_at,
        end_at: form.end_at, is_all_day: form.is_all_day, status: form.status, color: form.color,
        attendees: form.attendees.split(/[,;\s]+/).map((email) => email.trim()).filter(Boolean),
        scope: form.scope, recurrence_id: form.recurrence_id || null,
        attendees_can_invite_others: true, attendees_can_modify: false, attendees_can_see_other_attendees: true,
      };
      if (form.frequency) {
        payload.recurrence = { frequency: form.frequency, interval: form.interval, until: form.until };
        payload.recurrence_until = form.until;
      }
      if (editing) await api.put(`/agenda/events/${editing.id}`, payload);
      else await api.post('/agenda/events', payload);
      toast.success(editing ? 'Evento atualizado.' : 'Evento criado.');
      setEventModal(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar o evento', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function deleteEvent() {
    if (!editing) return;
    if (!window.confirm('Excluir este evento?')) return;
    try {
      await api.delete(`/agenda/events/${editing.id}`, { scope: form.scope, recurrence_id: form.recurrence_id || null });
      toast.success('Evento excluído.');
      setEventModal(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir o evento', errorMessage(error));
    }
  }

  async function saveCalendar() {
    setSaving(true);
    try {
      if (calendarEditing) {
        await api.put(`/agenda/calendars/${calendarEditing.id}`, calendarForm);
        if (calendarForm.email) await api.post(`/agenda/calendars/${calendarEditing.id}/shares`, { email: calendarForm.email, role: calendarForm.role });
      } else {
        await api.post('/agenda/calendars', calendarForm);
      }
      toast.success(calendarEditing ? 'Calendário atualizado.' : 'Calendário criado.');
      setCalendarModal(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar o calendário', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function toggleCalendar(id: number, checked: boolean) {
    setVisible((current) => checked ? [...current, id] : current.filter((item) => item !== id));
    try {
      await api.patch(`/agenda/calendars/${id}/visibility`, { is_visible: checked });
    } catch {
      // Compartilhamentos de leitura podem não permitir alterar a visibilidade.
    }
  }

  function move(offset: number) {
    const next = new Date(date);
    if (view === 'month') next.setMonth(next.getMonth() + offset);
    else if (view === 'week') next.setDate(next.getDate() + offset * 7);
    else next.setDate(next.getDate() + offset);
    setDate(next);
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Calendário"
        title="Agenda"
        description="Calendários, eventos, participantes, recorrências e importação ICS."
        actions={
          <>
            <Button renderIcon={Add} onClick={() => openCreate()}>Novo evento</Button>
            <OverflowMenu
              aria-label="Mais ações da agenda"
              iconDescription="Mais ações"
              renderIcon={OverflowMenuVertical}
              direction="bottom"
              flipped
            >
              <OverflowMenuItem itemText="Importar (.ics)" onClick={openImport} />
              <OverflowMenuItem itemText="Exportar (.ics)" onClick={() => window.open('/api/v1/agenda/ics', '_blank')} />
              <OverflowMenuItem itemText="Novo calendário" hasDivider onClick={openCalendarCreate} />
            </OverflowMenu>
          </>
        }
      />

      <div className="module-layout module-layout--agenda">
        <aside className="module-sidebar">
          <Tile className="module-sidebar__card">
            <h3>Meus calendários</h3>
            {calendars.map((calendar) => (
              <div key={calendar.id} className="calendar-row">
                <Checkbox
                  id={`cal-${calendar.id}`}
                  labelText={calendar.name}
                  checked={visible.includes(calendar.id)}
                  onChange={(_event, { checked }) => void toggleCalendar(calendar.id, checked)}
                />
                <div className="calendar-row__actions">
                  <span className="color-dot" style={{ background: colorHex(calendar.color, colors) }} />
                  {(calendar.role === 'owner' || calendar.role === 'writer') && (
                    <Button kind="ghost" size="xs" hasIconOnly renderIcon={Edit} iconDescription="Editar calendário" onClick={() => { setCalendarEditing(calendar); setCalendarForm({ name: calendar.name, description: '', color: calendar.color, timezone: calendar.timezone, email: '', role: 'reader' }); setCalendarModal(true); }} />
                  )}
                </div>
              </div>
            ))}
          </Tile>
        </aside>

        <div className="module-main">
          <Tile className="module-main__header">
            <div className="agenda-toolbar">
              <Button kind="ghost" onClick={() => move(-1)}>←</Button>
              <Button kind="ghost" onClick={() => setDate(new Date())}>Hoje</Button>
              <Button kind="ghost" onClick={() => move(1)}>→</Button>
              <strong>{date.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })}</strong>
            </div>
            <div className="module-toolbar">
              <Search size="sm" labelText="Buscar eventos" placeholder="Buscar eventos" value={search} onChange={(event) => setSearch(event.target.value)} onClear={() => setSearch('')} />
              <Select id="agenda-view" labelText="Visão" size="sm" value={view} onChange={(event) => setView(event.target.value as typeof view)}>
                <SelectItem value="month" text="Mês" />
                <SelectItem value="week" text="Semana" />
                <SelectItem value="day" text="Dia" />
                <SelectItem value="list" text="Agenda" />
              </Select>
            </div>
          </Tile>

          {loading ? <InlineLoading description="Carregando agenda…" /> : view === 'month' ? (
            <div className="agenda-month">
              {WEEKDAYS.map((weekday) => <div key={weekday} className="agenda-weekday">{weekday}</div>)}
              {days.map((day) => {
                const key = ymd(day);
                const items = byDay[key] ?? [];
                const outside = day.getMonth() !== date.getMonth();
                return (
                  <div key={key} className={`agenda-day ${outside ? 'agenda-day--outside' : ''} ${key === ymd(new Date()) ? 'agenda-day--today' : ''}`}>
                    <div className="agenda-day__head">
                      <button type="button" onClick={() => { setDate(day); setView('day'); }}>{day.getDate()}</button>
                      <Button kind="ghost" size="xs" hasIconOnly renderIcon={Add} iconDescription="Novo evento" onClick={() => openCreate(day)} />
                    </div>
                    {items.slice(0, 3).map((occurrence) => (
                      <button key={`${occurrence.id}-${occurrence.start}`} type="button" className={`agenda-chip ${occurrence.status === 'cancelled' ? 'is-cancelled' : ''}`} style={{ borderLeftColor: occurrence.color, background: `${occurrence.color}18` }} onClick={() => void openEdit(occurrence)}>
                        {!occurrence.is_all_day && <span>{new Date(occurrence.start).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}</span>}
                        {occurrence.title}
                      </button>
                    ))}
                    {items.length > 3 && <span className="agenda-more">+{items.length - 3} mais</span>}
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="agenda-list">
              {filtered.length === 0 && <p className="muted">Nenhum evento encontrado no período.</p>}
              {filtered.map((occurrence) => (
                <button key={`${occurrence.id}-${occurrence.start}`} type="button" className={`agenda-list-item ${occurrence.status === 'cancelled' ? 'is-cancelled' : ''}`} onClick={() => void openEdit(occurrence)}>
                  <span className="color-dot" style={{ background: occurrence.color }} />
                  <span className="agenda-list-item__date">{new Date(occurrence.start).toLocaleDateString('pt-BR')} {occurrence.is_all_day ? '· Dia inteiro' : `· ${new Date(occurrence.start).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}`}</span>
                  <strong>{occurrence.title}</strong>
                  <span>{occurrence.location ?? occurrence.calendar_name}</span>
                  {occurrence.is_recurring && <span className="task-badge">Recorrente</span>}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      <Modal
        open={eventModal}
        modalHeading={editing ? 'Editar evento' : 'Novo evento'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void saveEvent()}
        onRequestClose={() => setEventModal(false)}
        size="md"
      >
        <div className="modal-form">
          <TextInput id="event-title" labelText="Título" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
          <Select id="event-calendar" labelText="Calendário" value={String(form.calendar_id)} onChange={(event) => setForm({ ...form, calendar_id: Number(event.target.value) })}>
            {calendars.filter((calendar) => calendar.role !== 'reader' && calendar.role !== 'free_busy_reader').map((calendar) => <SelectItem key={calendar.id} value={String(calendar.id)} text={calendar.name} />)}
          </Select>
          <div className="form-row">
            <TextInput id="event-start" type="datetime-local" labelText="Início" value={form.start_at} onChange={(event) => setForm({ ...form, start_at: event.target.value })} />
            <TextInput id="event-end" type="datetime-local" labelText="Fim" value={form.end_at} onChange={(event) => setForm({ ...form, end_at: event.target.value })} />
          </div>
          <Checkbox id="event-allday" labelText="Dia inteiro" checked={form.is_all_day} onChange={(_event, { checked }) => setForm({ ...form, is_all_day: checked })} />
          <div className="form-row">
            <TextInput id="event-location" labelText="Local" value={form.location} onChange={(event) => setForm({ ...form, location: event.target.value })} />
            <TextInput id="event-meeting" labelText="URL da reunião" value={form.meeting_url} onChange={(event) => setForm({ ...form, meeting_url: event.target.value })} />
          </div>
          <TextArea id="event-description" labelText="Descrição" value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} rows={3} />
          <div className="form-row">
            <Select id="event-status" labelText="Status" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
              {Object.entries(STATUS_LABELS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
            </Select>
            <Select id="event-color" labelText="Cor" value={form.color} onChange={(event) => setForm({ ...form, color: event.target.value })}>
              <SelectItem value="" text="Herdar do calendário" />
              {Object.keys(colors).map((key) => <SelectItem key={key} value={key} text={key} />)}
            </Select>
          </div>
          <div className="form-row">
            <Select id="event-frequency" labelText="Repetição" value={form.frequency} onChange={(event) => setForm({ ...form, frequency: event.target.value })}>
              <SelectItem value="" text="Não se repete" />
              <SelectItem value="daily" text="Diariamente" />
              <SelectItem value="weekly" text="Semanalmente" />
              <SelectItem value="monthly" text="Mensalmente" />
              <SelectItem value="yearly" text="Anualmente" />
            </Select>
            <TextInput id="event-until" type="date" labelText="Repetir até" value={form.until} onChange={(event) => setForm({ ...form, until: event.target.value })} disabled={!form.frequency} />
          </div>
          <TextInput id="event-attendees" labelText="Participantes (e-mails separados por vírgula)" value={form.attendees} onChange={(event) => setForm({ ...form, attendees: event.target.value })} />
          {editing && editing.attendees.length > 0 && (
            <div>
              <p className="field-label">Participantes atuais</p>
              {editing.attendees.map((attendee) => (
                <div key={attendee.email} className="sharing-row"><span>{attendee.name ?? attendee.email}</span><span>{attendee.status}</span></div>
              ))}
            </div>
          )}
          {editing?.recurrence && (
            <Select id="event-scope" labelText="Aplicar alteração" value={form.scope} onChange={(event) => setForm({ ...form, scope: event.target.value })}>
              <SelectItem value="all" text="Todos os eventos" />
              <SelectItem value="this" text="Somente esta ocorrência" />
              <SelectItem value="following" text="Esta e as seguintes" />
            </Select>
          )}
          {editing && <Button kind="danger--ghost" renderIcon={TrashCan} onClick={() => void deleteEvent()}>Excluir evento</Button>}
        </div>
      </Modal>

      <Modal
        open={calendarModal}
        modalHeading={calendarEditing ? 'Editar calendário' : 'Novo calendário'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void saveCalendar()}
        onRequestClose={() => setCalendarModal(false)}
        size="sm"
      >
        <div className="modal-form">
          <TextInput id="cal-name" labelText="Nome" value={calendarForm.name} onChange={(event) => setCalendarForm({ ...calendarForm, name: event.target.value })} required />
          <TextArea id="cal-description" labelText="Descrição" value={calendarForm.description} onChange={(event) => setCalendarForm({ ...calendarForm, description: event.target.value })} rows={2} />
          <Select id="cal-color" labelText="Cor" value={calendarForm.color} onChange={(event) => setCalendarForm({ ...calendarForm, color: event.target.value })}>
            {Object.keys(colors).map((key) => <SelectItem key={key} value={key} text={key} />)}
          </Select>
          <TextInput id="cal-timezone" labelText="Fuso horário" value={calendarForm.timezone} onChange={(event) => setCalendarForm({ ...calendarForm, timezone: event.target.value })} />
          {calendarEditing && calendarEditing.role === 'owner' && (
            <>
              <h4>Compartilhar</h4>
              <TextInput id="cal-email" labelText="E-mail do usuário" value={calendarForm.email} onChange={(event) => setCalendarForm({ ...calendarForm, email: event.target.value })} />
              <Select id="cal-role" labelText="Permissão" value={calendarForm.role} onChange={(event) => setCalendarForm({ ...calendarForm, role: event.target.value })}>
                <SelectItem value="writer" text="Pode editar" />
                <SelectItem value="reader" text="Pode ver detalhes" />
                <SelectItem value="free_busy_reader" text="Vê apenas disponibilidade" />
              </Select>
              {calendarEditing.shares.map((share) => <div key={share.id} className="sharing-row"><span>{share.name} · {share.email}</span><span>{share.role}</span></div>)}
              <Button kind="danger--ghost" renderIcon={TrashCan} onClick={async () => { if (!window.confirm(`Excluir o calendário “${calendarEditing.name}” e todos os eventos?`)) return; await api.delete(`/agenda/calendars/${calendarEditing.id}`); setCalendarModal(false); toast.success('Calendário excluído.'); await load(); }}>Excluir calendário</Button>
            </>
          )}
        </div>
      </Modal>

      <Modal
        open={importModal}
        modalHeading="Importar agenda (.ics)"
        primaryButtonText={importing ? 'Importando…' : 'Importar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={importing || !importFile}
        onRequestSubmit={() => void importICS()}
        onRequestClose={() => { setImportModal(false); setImportFile(null); }}
        size="sm"
      >
        <div className="modal-form">
          <p className="muted small">Importe eventos de um arquivo .ics para um calendário de sua escolha.</p>
          <Select
            id="ics-calendar"
            labelText="Calendário de destino"
            value={String(importCalendar || calendars[0]?.id || '')}
            onChange={(event) => setImportCalendar(Number(event.target.value))}
          >
            {calendars.filter((calendar) => calendar.role !== 'reader' && calendar.role !== 'free_busy_reader').map((calendar) => (
              <SelectItem key={calendar.id} value={String(calendar.id)} text={calendar.name} />
            ))}
          </Select>
          <TextInput
            key={String(importModal)}
            id="ics"
            type="file"
            labelText="Arquivo .ics"
            accept=".ics,.ical,text/calendar"
            onChange={(event) => setImportFile(event.target.files?.[0] ?? null)}
          />
        </div>
      </Modal>
    </div>
  );
}
