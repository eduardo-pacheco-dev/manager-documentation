import { Modal } from '@carbon/react';

interface ConfirmModalProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmModal({
  open,
  title,
  message,
  confirmLabel = 'Confirmar',
  cancelLabel = 'Cancelar',
  danger = true,
  loading = false,
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  return (
    <Modal
      open={open}
      modalHeading={title}
      primaryButtonText={loading ? 'Processando…' : confirmLabel}
      secondaryButtonText={cancelLabel}
      danger={danger}
      primaryButtonDisabled={loading}
      onRequestSubmit={onConfirm}
      onRequestClose={onCancel}
      onSecondarySubmit={onCancel}
      size="sm"
    >
      <p>{message}</p>
    </Modal>
  );
}
