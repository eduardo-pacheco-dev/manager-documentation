import { useCallback, useEffect, useRef, useState } from 'react';
import { DataBase, OverflowMenuVertical, Save, Upload } from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  InlineLoading,
  InlineNotification,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  Tag,
  TextInput,
  Toggle,
} from '@carbon/react';
import { ConfirmModal } from './ConfirmModal';
import { EmptyState } from './Feedback';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage, fullDate } from '../lib/format';

interface BackupEntry {
  name: string;
  size: number;
  created_at: string;
}

interface BackupSettings {
  directory: string;
  retention: number;
  automatic: boolean;
  hour: number;
  minute: number;
  weekdays: number[];
  next_run?: string;
  uploads_dir: string;
}

interface ListResponse {
  data: BackupEntry[];
  settings: BackupSettings;
}

type PendingAction = { type: 'delete'; entry: BackupEntry } | { type: 'restore'; entry: BackupEntry };

const WEEKDAYS = [
  { value: 1, label: 'Seg' },
  { value: 2, label: 'Ter' },
  { value: 3, label: 'Qua' },
  { value: 4, label: 'Qui' },
  { value: 5, label: 'Sex' },
  { value: 6, label: 'Sáb' },
  { value: 7, label: 'Dom' },
];

function formatScheduleTime(hour: number, minute: number): string {
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / 1024 ** index;
  return `${value.toLocaleString('pt-BR', { maximumFractionDigits: index === 0 ? 0 : 1 })} ${units[index]}`;
}

export function BackupsPanel() {
  const toast = useToast();
  const [entries, setEntries] = useState<BackupEntry[]>([]);
  const [settings, setSettings] = useState<BackupSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [pending, setPending] = useState<PendingAction | null>(null);
  const [restoreConfirm, setRestoreConfirm] = useState('');
  const [working, setWorking] = useState(false);
  const [pendingImport, setPendingImport] = useState<File | null>(null);
  const [importRestore, setImportRestore] = useState(false);
  const [importing, setImporting] = useState(false);
  const [scheduleEnabled, setScheduleEnabled] = useState(true);
  const [scheduleTime, setScheduleTime] = useState('03:00');
  const [scheduleDays, setScheduleDays] = useState<number[]>([1, 2, 3, 4, 5, 6, 7]);
  const [savingSchedule, setSavingSchedule] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    try {
      const response = await api.get<ListResponse>('/backups');
      setEntries(response.data ?? []);
      setSettings(response.settings);
      if (response.settings) {
        setScheduleEnabled(response.settings.automatic);
        setScheduleTime(formatScheduleTime(response.settings.hour, response.settings.minute));
        setScheduleDays(response.settings.weekdays?.length ? response.settings.weekdays : [1, 2, 3, 4, 5, 6, 7]);
      }
    } catch (error) {
      toast.error('Não foi possível carregar os backups', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  async function createBackup() {
    setCreating(true);
    try {
      const response = await api.post<{ message: string }>('/backups');
      toast.success('Backup criado', response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível criar o backup', errorMessage(error));
    } finally {
      setCreating(false);
    }
  }

  async function importBackup() {
    if (!pendingImport) return;
    setImporting(true);
    try {
      const form = new FormData();
      form.append('file', pendingImport);
      if (importRestore) form.append('restore', 'true');
      const response = await api.upload<{ message: string }>('/backups/import', form);
      toast.success('Backup importado', response.message);
      setPendingImport(null);
      setImportRestore(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível importar o backup', errorMessage(error));
    } finally {
      setImporting(false);
    }
  }

  async function saveSchedule() {
    const [hourText, minuteText] = scheduleTime.split(':');
    const hour = Number(hourText);
    const minute = Number(minuteText);
    setSavingSchedule(true);
    try {
      const response = await api.put<{ message: string; settings: BackupSettings }>('/backups/schedule', {
        enabled: scheduleEnabled,
        hour,
        minute,
        weekdays: scheduleDays,
      });
      setSettings(response.settings);
      toast.success('Agendamento salvo', response.message);
    } catch (error) {
      toast.error('Não foi possível salvar o agendamento', errorMessage(error));
    } finally {
      setSavingSchedule(false);
    }
  }

  function toggleDay(day: number, checked: boolean) {
    setScheduleDays((current) => {
      if (checked) {
        return [...new Set([...current, day])].sort((left, right) => left - right);
      }
      return current.filter((value) => value !== day);
    });
  }

  async function confirmAction() {
    if (!pending) return;
    setWorking(true);
    try {
      if (pending.type === 'delete') {
        await api.delete(`/backups/${encodeURIComponent(pending.entry.name)}`);
        toast.success('Backup removido');
      } else {
        const response = await api.post<{ message: string }>(
          `/backups/${encodeURIComponent(pending.entry.name)}/restore`,
          { confirm: restoreConfirm.trim() },
        );
        toast.success('Backup restaurado', response.message);
      }
      setPending(null);
      setRestoreConfirm('');
      await load();
    } catch (error) {
      toast.error('Não foi possível concluir a operação', errorMessage(error));
    } finally {
      setWorking(false);
    }
  }

  function downloadBackup(entry: BackupEntry) {
    const link = document.createElement('a');
    link.href = `/api/v1/backups/${encodeURIComponent(entry.name)}/download`;
    link.download = entry.name;
    document.body.appendChild(link);
    link.click();
    link.remove();
  }

  const daysLabel = settings
    ? WEEKDAYS.filter((day) => settings.weekdays?.includes(day.value)).map((day) => day.label).join(', ')
    : '';

  return (
    <div className="backups-panel">
      <input
        ref={fileInput}
        type="file"
        accept=".zip,application/zip"
        hidden
        onChange={(event) => {
          const file = event.target.files?.[0] ?? null;
          event.target.value = '';
          if (file) {
            setImportRestore(false);
            setPendingImport(file);
          }
        }}
      />
      <div className="section-actions section-actions--start">
        <Button kind="tertiary" renderIcon={Upload} onClick={() => fileInput.current?.click()} disabled={importing}>
          {importing ? 'Importando…' : 'Importar .zip'}
        </Button>
        <Button renderIcon={DataBase} onClick={() => void createBackup()} disabled={creating || importing}>
          {creating ? 'Gerando backup…' : 'Criar backup agora'}
        </Button>
      </div>

      <div className="backup-schedule">
        <div className="backup-schedule__row">
          <Toggle
            id="backup-schedule-enabled"
            labelText="Backup automático"
            labelA="Desativado"
            labelB="Ativado"
            toggled={scheduleEnabled}
            onToggle={(checked) => setScheduleEnabled(checked)}
          />
          <TextInput
            id="backup-schedule-time"
            type="time"
            labelText="Horário"
            value={scheduleTime}
            onChange={(event) => setScheduleTime(event.target.value)}
          />
        </div>
        <fieldset className="backup-schedule__days">
          <legend>Dias da semana</legend>
          {WEEKDAYS.map((day) => (
            <Checkbox
              key={day.value}
              id={`backup-weekday-${day.value}`}
              labelText={day.label}
              checked={scheduleDays.includes(day.value)}
              onChange={(_event, { checked }) => toggleDay(day.value, checked)}
            />
          ))}
        </fieldset>
        <div className="section-actions section-actions--start">
          <Button
            renderIcon={Save}
            onClick={() => void saveSchedule()}
            disabled={savingSchedule || (scheduleEnabled && scheduleDays.length === 0)}
          >
            {savingSchedule ? 'Salvando…' : 'Salvar agendamento'}
          </Button>
        </div>
      </div>

      {settings && (
        <InlineNotification
          kind="info"
          lowContrast
          hideCloseButton
          title={`Retenção de ${settings.retention} backup(s)`}
          subtitle={
            `${settings.automatic
              ? `Um backup automático é gerado${daysLabel ? ` (${daysLabel})` : ''} às ${formatScheduleTime(settings.hour, settings.minute)}.`
              : 'O backup automático está desativado; use o botão acima para gerar manualmente.'} ` +
            `Os arquivos ficam em ${settings.directory} e incluem banco, uploads locais e anexos remotos (S3/R2), quando configurados.`
          }
        />
      )}

      {creating && (
        <InlineLoading description="Compactando banco de dados e uploads. Isso pode levar alguns minutos…" />
      )}
      {importing && (
        <InlineLoading description="Enviando e validando o arquivo de backup…" />
      )}

      {loading ? (
        <InlineLoading description="Carregando backups…" />
      ) : entries.length === 0 ? (
        <EmptyState
          icon={DataBase}
          title="Nenhum backup encontrado"
          description="Crie o primeiro backup para proteger os dados do sistema."
          action={<Button renderIcon={DataBase} onClick={() => void createBackup()} disabled={creating}>Criar backup</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de backups">
              <thead>
                <tr>
                  <th>Arquivo</th>
                  <th className="hide-sm">Criado em</th>
                  <th>Tamanho</th>
                  <th className="cell-actions">Ações</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.name}>
                    <td>
                      <div className="user-cell">
                        <span className="cell-primary">{entry.name}</span>
                        <span className="cell-secondary">Banco de dados + uploads</span>
                      </div>
                    </td>
                    <td className="hide-sm">{fullDate(entry.created_at)}</td>
                    <td>
                      <Tag type="gray" size="sm">{formatBytes(entry.size)}</Tag>
                    </td>
                    <td className="cell-actions">
                      <OverflowMenu
                        aria-label={`Ações de ${entry.name}`}
                        iconDescription={`Ações de ${entry.name}`}
                        renderIcon={OverflowMenuVertical}
                        size="sm"
                        flipped
                      >
                        <OverflowMenuItem itemText="Baixar" onClick={() => downloadBackup(entry)} />
                        <OverflowMenuItem
                          itemText="Restaurar"
                          onClick={() => {
                            setRestoreConfirm('');
                            setPending({ type: 'restore', entry });
                          }}
                        />
                        <OverflowMenuItem
                          itemText="Excluir"
                          isDelete
                          hasDivider
                          onClick={() => setPending({ type: 'delete', entry })}
                        />
                      </OverflowMenu>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer">
            <span className="table-footer__order">
              {settings?.automatic && settings.next_run
                ? `Próximo backup automático: ${fullDate(settings.next_run)}`
                : 'Backup automático desativado'}
            </span>
          </div>
        </div>
      )}

      <Modal
        open={pendingImport !== null}
        modalHeading="Importar backup"
        primaryButtonText={importing ? 'Importando…' : 'Importar backup'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={importing || pendingImport === null}
        onRequestSubmit={() => void importBackup()}
        onRequestClose={() => {
          if (!importing) setPendingImport(null);
        }}
        onSecondarySubmit={() => setPendingImport(null)}
        size="sm"
      >
        <p>
          Envie um arquivo <code>.zip</code> gerado pelo Documentation Center. O arquivo precisa conter o
          <code> database.sql</code>; os uploads locais são opcionais.
        </p>
        {pendingImport && (
          <p>
            <strong>{pendingImport.name}</strong> ({formatBytes(pendingImport.size)})
          </p>
        )}
        <Checkbox
          id="import-restore"
          labelText="Restaurar imediatamente após importar"
          checked={importRestore}
          disabled={importing}
          onChange={(_event, { checked }) => setImportRestore(checked)}
        />
        {importRestore && (
          <InlineNotification
            kind="warning"
            lowContrast
            hideCloseButton
            title="Restauração imediata"
            subtitle="O banco de dados e os uploads atuais serão substituídos pelo conteúdo do arquivo importado."
          />
        )}
      </Modal>

      <ConfirmModal
        open={pending?.type === 'delete'}
        title="Excluir backup"
        message={pending ? `Remover permanentemente o arquivo ${pending.entry.name}? Esta ação não pode ser desfeita.` : ''}
        confirmLabel="Excluir"
        loading={working}
        onCancel={() => setPending(null)}
        onConfirm={() => void confirmAction()}
      />

      <Modal
        open={pending?.type === 'restore'}
        modalHeading="Restaurar backup"
        primaryButtonText={working ? 'Restaurando…' : 'Restaurar backup'}
        secondaryButtonText="Cancelar"
        danger
        primaryButtonDisabled={working || !pending || restoreConfirm.trim() !== pending.entry.name}
        onRequestSubmit={() => void confirmAction()}
        onRequestClose={() => setPending(null)}
        onSecondarySubmit={() => setPending(null)}
        size="sm"
      >
        <p>
          A restauração substitui os dados atuais do banco e os uploads locais pelo conteúdo de{' '}
          <strong>{pending?.entry.name}</strong>. As sessões ativas e alterações feitas depois do backup serão perdidas.
        </p>
        <TextInput
          id="restore-confirm"
          labelText={`Digite ${pending?.entry.name ?? ''} para confirmar`}
          value={restoreConfirm}
          onChange={(event) => setRestoreConfirm(event.target.value)}
          disabled={working}
        />
      </Modal>
    </div>
  );
}
