import { useEffect, useState } from 'react';
import { Button, InlineLoading, Modal, PasswordInput, TextInput } from '@carbon/react';
import { Copy } from '@carbon/icons-react';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { IepmsUser } from '../lib/types';

interface Props {
  open: boolean;
  user: IepmsUser | null;
  onClose: () => void;
}

export function IepmsPasswordModal({ open, user, onClose }: Props) {
  const toast = useToast();
  const [password, setPassword] = useState('');
  const [revealed, setRevealed] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) return;
    setPassword('');
    setRevealed('');
    setError('');
    setLoading(false);
  }, [open, user]);

  async function submit() {
    if (!user) return;
    if (revealed) {
      onClose();
      return;
    }
    if (!password.trim()) {
      setError('Informe a sua senha para visualizar.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const response = await api.post<{ data: { password: string } }>(`/iepms-users/${user.id}/password`, {
        password,
      });
      setRevealed(response.data.password);
      setPassword('');
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.fields.password ?? err.message);
      } else {
        setError(errorMessage(err));
      }
    } finally {
      setLoading(false);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(revealed);
      toast.success('Senha copiada');
    } catch {
      toast.error('Não foi possível copiar', 'Copie manualmente o valor exibido.');
    }
  }

  return (
    <Modal
      open={open}
      modalHeading={user ? `Senha de ${user.name}` : 'Visualizar senha'}
      primaryButtonText={loading ? 'Verificando…' : revealed ? 'Fechar' : 'Visualizar'}
      secondaryButtonText={revealed ? undefined : 'Cancelar'}
      primaryButtonDisabled={loading}
      onRequestSubmit={() => void submit()}
      onRequestClose={onClose}
      onSecondarySubmit={onClose}
      size="sm"
    >
      {user && (
        <div className="modal-form">
          <p>
            Para visualizar a senha de <strong>{user.name}</strong> ({user.resource_id}), confirme a
            senha da sua própria conta.
          </p>
          {revealed ? (
            <>
              <TextInput
                id="iepms-revealed-password"
                labelText="Senha"
                value={revealed}
                readOnly
              />
              <div className="section-actions">
                <Button kind="tertiary" size="sm" renderIcon={Copy} onClick={() => void copy()}>
                  Copiar senha
                </Button>
              </div>
            </>
          ) : (
            <>
              <PasswordInput
                id="iepms-confirm-password"
                labelText="Sua senha"
                value={password}
                onChange={(event) => {
                  setPassword(event.target.value);
                  setError('');
                }}
                invalid={Boolean(error)}
                invalidText={error}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') void submit();
                }}
              />
              {loading && <InlineLoading description="Verificando senha…" />}
            </>
          )}
        </div>
      )}
    </Modal>
  );
}
