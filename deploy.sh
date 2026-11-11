#!/usr/bin/env bash
#
# Documentation Center - VPS deployment helper (PM2)
#
# Uso:
#   sudo bash deploy.sh setup --url https://docs.exemplo.com --admin-email admin@exemplo.com
#   ./deploy.sh --artifact /tmp/documentation-center-linux-amd64.tar.gz --tag 1a2b3c4d
#   ./deploy.sh rollback
#
# O modo "setup" deve ser executado uma única vez na VPS. O modo de deploy é
# chamado pelo GitHub Actions (ou manualmente) após o build do binário Go e do
# frontend. O PM2 mantém o processo ativo e o reinicia no boot via systemd.

set -Eeuo pipefail

APP_NAME="${PM2_APP_NAME:-documentation-center}"
APP_DIR="${APP_DIR:-}"
APP_USER="${APP_USER:-}"
APP_PORT="${APP_PORT:-8080}"
APP_BIND="${APP_BIND:-0.0.0.0}"
HEALTH_PATH="${HEALTH_PATH:-/healthz}"
KEEP_RELEASES="${KEEP_RELEASES:-5}"
MODE="deploy"
ARTIFACT=""
TAG=""
PUBLIC_URL_INPUT="${PUBLIC_URL:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
ADMIN_NAME="${ADMIN_NAME:-Administrador}"
DB_NAME="${DB_NAME:-documentation_center_go}"
DB_USER="${DB_USER:-documentation}"
DB_PASSWORD="${DB_PASSWORD:-}"
KEEP_ARTIFACT=0
SKIP_HEALTH=0

log()  { printf '\033[1;34m[deploy]\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m[deploy]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[deploy]\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
	cat <<'EOF'
Documentation Center - deploy com PM2

Modo setup (uma vez, como root/sudo):
  sudo bash deploy.sh setup [opções]

  --url URL                 URL pública (ex.: https://docs.exemplo.com)
  --admin-email EMAIL       E-mail do administrador inicial
  --admin-password SENHA    Senha do administrador (gerada se omitida)
  --admin-name NOME         Nome do administrador (padrão: Administrador)
  --dir DIR                 Diretório da aplicação (padrão: /opt/documentation-center)
  --user USER               Usuário Linux que executará o PM2 (padrão: SUDO_USER)
  --port PORTA              Porta HTTP (padrão: 8080)
  --bind ENDEREÇO           Endereço de bind (padrão: 0.0.0.0)

  Variáveis opcionais do MySQL: DB_NAME (documentation_center_go), DB_USER
  (documentation), DB_PASSWORD (gerada) e MYSQL_ROOT_PASSWORD (quando o root
  não usa autenticação por socket).

Deploy de um artefato gerado pelo GitHub Actions:
  ./deploy.sh --artifact ARQUIVO.tar.gz --tag TAG [--port 8080]

  --artifact ARQUIVO        Pacote com "server" e "static/"
  --tag TAG                 Identificador da release (padrão: data/hora)
  --dir DIR                 Diretório da aplicação
  --port PORTA              Porta usada pelo PM2 e pelo health check
  --no-health               Não aguarda o health check
  --keep-artifact           Mantém o tar.gz após o deploy

Rollback para a release anterior:
  ./deploy.sh rollback [--dir DIR]
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		setup) MODE="setup"; shift ;;
		rollback) MODE="rollback"; shift ;;
		--artifact) ARTIFACT="${2:?Informe o arquivo}"; shift 2 ;;
		--tag) TAG="${2:?Informe a tag}"; shift 2 ;;
		--dir) APP_DIR="${2:?Informe o diretório}"; shift 2 ;;
		--user) APP_USER="${2:?Informe o usuário}"; shift 2 ;;
		--port) APP_PORT="${2:?Informe a porta}"; shift 2 ;;
		--bind) APP_BIND="${2:?Informe o endereço}"; shift 2 ;;
		--url) PUBLIC_URL_INPUT="${2:?Informe a URL}"; shift 2 ;;
		--admin-email) ADMIN_EMAIL="${2:?Informe o e-mail}"; shift 2 ;;
		--admin-password) ADMIN_PASSWORD="${2:?Informe a senha}"; shift 2 ;;
		--admin-name) ADMIN_NAME="${2:?Informe o nome}"; shift 2 ;;
		--no-health) SKIP_HEALTH=1; shift ;;
		--keep-artifact) KEEP_ARTIFACT=1; shift ;;
		-h|--help) usage; exit 0 ;;
		*) die "Opção desconhecida: $1 (use --help)" ;;
	esac
done

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

if [[ -z "$APP_DIR" ]]; then
	if [[ -d "$SCRIPT_DIR/.git" || -f "$SCRIPT_DIR/ecosystem.config.cjs" ]]; then
		APP_DIR="$SCRIPT_DIR"
	else
		APP_DIR="/opt/documentation-center"
	fi
fi

if [[ -z "$APP_USER" ]]; then
	APP_USER="${SUDO_USER:-$(id -un)}"
fi
APP_HOME="$(getent passwd "$APP_USER" 2>/dev/null | cut -d: -f6 || true)"
[[ -n "$APP_HOME" ]] || APP_HOME="/home/$APP_USER"

run_as_app() {
	if [[ "$(id -un)" == "$APP_USER" ]]; then
		"$@"
	elif command -v sudo >/dev/null 2>&1; then
		sudo -u "$APP_USER" -H "$@"
	elif command -v runuser >/dev/null 2>&1; then
		runuser -u "$APP_USER" -- "$@"
	else
		die "Não foi possível executar como $APP_USER (instale sudo)."
	fi
}

install_packages() {
	if command -v apt-get >/dev/null 2>&1; then
		apt-get update -y
		DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl gnupg git openssl tar sudo
	elif command -v dnf >/dev/null 2>&1; then
		dnf install -y ca-certificates curl gnupg git openssl tar sudo
	elif command -v yum >/dev/null 2>&1; then
		yum install -y ca-certificates curl gnupg git openssl tar sudo
	else
		warn "Gerenciador de pacotes não reconhecido; verifique curl, git, openssl, tar e sudo."
	fi
}

install_mysql_server() {
	if command -v mysql >/dev/null 2>&1 && { command -v mysqld >/dev/null 2>&1 || systemctl list-unit-files 2>/dev/null | grep -qE '^(mysql|mariadb)\.service'; }; then
		log "MySQL já instalado."
	else
		log "Instalando MySQL..."
		if command -v apt-get >/dev/null 2>&1; then
			DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server || \
				DEBIAN_FRONTEND=noninteractive apt-get install -y mariadb-server
		elif command -v dnf >/dev/null 2>&1; then
			dnf install -y mysql-server || dnf install -y mariadb-server
		elif command -v yum >/dev/null 2>&1; then
			yum install -y mysql-server || yum install -y mariadb-server
		else
			die "Instale o MySQL 8+ manualmente e execute novamente."
		fi
	fi
	if systemctl list-unit-files 2>/dev/null | grep -q '^mysql.service'; then
		systemctl enable --now mysql
	elif systemctl list-unit-files 2>/dev/null | grep -q '^mariadb.service'; then
		systemctl enable --now mariadb
	fi
	for _ in $(seq 1 30); do
		if mysql_exec -e 'SELECT 1' >/dev/null 2>&1; then
			break
		fi
		sleep 1
	done
}

mysql_exec() {
	if mysql --protocol=socket -uroot -e 'SELECT 1' >/dev/null 2>&1; then
		mysql --protocol=socket -uroot "$@"
	else
		mysql -uroot -p"${MYSQL_ROOT_PASSWORD:-}" "$@"
	fi
}

read_env_value() {
	local key="$1"
	[[ -f "$APP_DIR/shared/.env" ]] || return 0
	grep -E "^${key}=" "$APP_DIR/shared/.env" | head -n1 | cut -d= -f2- || true
}

provision_mysql() {
	log "Configurando banco MySQL..."
	local existing_name existing_user existing_password
	existing_name="$(read_env_value DB_NAME)"
	existing_user="$(read_env_value DB_USER)"
	existing_password="$(read_env_value DB_PASSWORD)"
	if [[ -n "$existing_name" ]]; then DB_NAME="$existing_name"; fi
	if [[ -n "$existing_user" ]]; then DB_USER="$existing_user"; fi
	if [[ -z "$DB_NAME" ]]; then DB_NAME="documentation_center_go"; fi
	if [[ -z "$DB_USER" ]]; then DB_USER="documentation"; fi
	if [[ -z "$DB_PASSWORD" ]]; then DB_PASSWORD="$existing_password"; fi
	if [[ -z "$DB_PASSWORD" ]]; then DB_PASSWORD="$(gen_password)"; fi

	local password_sql="${DB_PASSWORD//\'/\'\'}"
	local sql
	sql=$(cat <<EOF
CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${password_sql}';
CREATE USER IF NOT EXISTS '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${password_sql}';
ALTER USER '${DB_USER}'@'localhost' IDENTIFIED BY '${password_sql}';
ALTER USER '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${password_sql}';
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'127.0.0.1';
FLUSH PRIVILEGES;
EOF
)
	if ! printf '%s\n' "$sql" | mysql_exec >/dev/null 2>&1; then
		die "Não foi possível configurar o MySQL. Verifique se o root usa autenticação por socket ou defina MYSQL_ROOT_PASSWORD."
	fi
	log "Banco ${DB_NAME} e usuário ${DB_USER} prontos."
}

install_node_pm2() {
	local node_major=0
	if command -v node >/dev/null 2>&1; then
		node_major="$(node -p "Number(process.versions.node.split('.')[0])" 2>/dev/null || echo 0)"
	fi
	if [[ "$node_major" -lt 18 ]]; then
		log "Instalando Node.js 22..."
		if command -v apt-get >/dev/null 2>&1; then
			curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
			DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
		elif command -v dnf >/dev/null 2>&1; then
			curl -fsSL https://rpm.nodesource.com/setup_22.x | bash -
			dnf install -y nodejs
		elif command -v yum >/dev/null 2>&1; then
			curl -fsSL https://rpm.nodesource.com/setup_22.x | bash -
			yum install -y nodejs
		else
			die "Instale o Node.js 18+ manualmente e execute novamente."
		fi
	fi
	node --version
	if ! command -v pm2 >/dev/null 2>&1; then
		log "Instalando PM2..."
		npm install -g pm2
	fi
	pm2 --version
}

gen_key() {
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -hex 32
	else
		head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
	fi
}

gen_password() {
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -base64 32 | tr -dc 'A-Za-z0-9' | head -c 20
	else
		head -c 32 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 20
	fi
}

write_ecosystem() {
	local file="$1"
	cat > "$file" <<EOF
module.exports = {
  apps: [
    {
      name: '${APP_NAME}',
      cwd: '${APP_DIR}/current',
      script: '${APP_DIR}/current/server',
      interpreter: 'none',
      exec_mode: 'fork',
      instances: 1,
      autorestart: true,
      restart_delay: 3000,
      max_restarts: 100,
      max_memory_restart: '512M',
      kill_timeout: 5000,
      time: true,
      merge_logs: true,
      out_file: '${APP_DIR}/shared/logs/out.log',
      error_file: '${APP_DIR}/shared/logs/error.log',
      env: {
        NODE_ENV: 'production',
        ADDR: '${APP_BIND}:${APP_PORT}'
      }
    }
  ]
};
EOF
	chown "$APP_USER":"$APP_USER" "$file" 2>/dev/null || true
}

setup() {
	[[ "$(id -u)" -eq 0 ]] || die "Execute o setup como root: sudo bash deploy.sh setup ..."
	if [[ "$APP_USER" == "root" ]]; then
		warn "O PM2 será executado como root. Para maior isolamento, crie um usuário de deploy e use --user."
	fi

	if [[ -z "$PUBLIC_URL_INPUT" ]]; then
		local ip
		ip="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
		PUBLIC_URL_INPUT="http://${ip:-localhost}:${APP_PORT}"
		warn "PUBLIC_URL não informada; usando ${PUBLIC_URL_INPUT}. Configure um domínio/HTTPS depois."
	fi
	PUBLIC_URL_INPUT="${PUBLIC_URL_INPUT%/}"

	local secure_cookies="false"
	case "$PUBLIC_URL_INPUT" in
		https://*) secure_cookies="true" ;;
	esac

	install_packages
	install_mysql_server
	install_node_pm2

	log "Preparando ${APP_DIR}..."
	mkdir -p "$APP_DIR/shared/data" "$APP_DIR/shared/logs" "$APP_DIR/releases"
	chown -R "$APP_USER":"$APP_USER" "$APP_DIR"

	provision_mysql

	local env_created=0
	if [[ ! -f "$APP_DIR/shared/.env" ]]; then
		env_created=1
		[[ -n "$ADMIN_PASSWORD" ]] || ADMIN_PASSWORD="$(gen_password)"
		local app_key
		app_key="$(gen_key)"
		cat > "$APP_DIR/shared/.env" <<EOF
APP_ENV=production
ADDR=${APP_BIND}:${APP_PORT}
PUBLIC_URL=${PUBLIC_URL_INPUT}
ALLOWED_ORIGINS=${PUBLIC_URL_INPUT}
STATIC_DIR=./static
SECURE_COOKIES=${secure_cookies}
APP_TIMEZONE=America/Sao_Paulo
APP_KEY=${app_key}

UPLOAD_DIR=${APP_DIR}/shared/data/uploads
UPLOAD_MAX_MB=40
SESSION_TTL_HOURS=168

DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}

SEED_ADMIN_NAME=${ADMIN_NAME}
SEED_ADMIN_EMAIL=${ADMIN_EMAIL}
SEED_ADMIN_PASSWORD=${ADMIN_PASSWORD}
SEED_DEMO_DATA=false

SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=no-reply@localhost
EOF
		chown "$APP_USER":"$APP_USER" "$APP_DIR/shared/.env"
		chmod 600 "$APP_DIR/shared/.env"
		printf '%s\n' "$ADMIN_PASSWORD" > "$APP_DIR/shared/INITIAL-ADMIN-PASSWORD.txt"
		chown "$APP_USER":"$APP_USER" "$APP_DIR/shared/INITIAL-ADMIN-PASSWORD.txt"
		chmod 600 "$APP_DIR/shared/INITIAL-ADMIN-PASSWORD.txt"
		log "Arquivo .env criado em $APP_DIR/shared/.env"
	else
		log "Mantendo .env existente em $APP_DIR/shared/.env"
		if ! grep -q '^DB_HOST=' "$APP_DIR/shared/.env"; then
			cat >> "$APP_DIR/shared/.env" <<EOF

DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
EOF
			log "Variáveis do MySQL adicionadas ao .env existente."
		fi
	fi

	write_ecosystem "$APP_DIR/ecosystem.config.cjs"
	if [[ -f "$SCRIPT_DIR/deploy.sh" && "$SCRIPT_DIR/deploy.sh" != "$APP_DIR/deploy.sh" ]]; then
		cp "$SCRIPT_DIR/deploy.sh" "$APP_DIR/deploy.sh"
		chown "$APP_USER":"$APP_USER" "$APP_DIR/deploy.sh"
		chmod +x "$APP_DIR/deploy.sh"
	fi

	log "Configurando PM2 no boot (systemd)..."
	local startup_output startup_cmd
	startup_output="$(env PATH="$PATH:/usr/bin" pm2 startup systemd -u "$APP_USER" --hp "$APP_HOME" 2>&1 || true)"
	printf '%s\n' "$startup_output" > /tmp/pm2-startup.log
	startup_cmd="$(printf '%s\n' "$startup_output" | grep -E '^sudo ' | tail -n1 || true)"
	if [[ -n "$startup_cmd" ]]; then
		log "Executando comando de startup do PM2..."
		bash -c "$startup_cmd" || true
	fi
	if [[ -f "/etc/systemd/system/pm2-$APP_USER.service" ]]; then
		log "PM2 startup configurado."
	else
		warn "PM2 startup não confirmado. Verifique:"
		sed 's/^/  /' /tmp/pm2-startup.log >&2 || true
		warn "Rode manualmente o comando 'sudo env PATH=... pm2 startup systemd -u $APP_USER --hp $APP_HOME'."
	fi
	run_as_app pm2 save >/dev/null 2>&1 || true

	cat > "$APP_DIR/DEPLOY-INFO.txt" <<EOF
Documentation Center - informações da VPS
Diretório: ${APP_DIR}
Usuário do PM2: ${APP_USER}
Porta: ${APP_PORT}
URL pública: ${PUBLIC_URL_INPUT}
Banco MySQL: ${DB_NAME} (usuário ${DB_USER})
Env: ${APP_DIR}/shared/.env
Uploads: ${APP_DIR}/shared/data/uploads
Logs: ${APP_DIR}/shared/logs/
Backup: mysqldump --single-transaction -u ${DB_USER} -p ${DB_NAME} > backup.sql

Próximos passos:
1. Configure os secrets do GitHub: VPS_HOST, VPS_USER (${APP_USER}), VPS_SSH_KEY
   e, se necessário, VPS_PORT e VPS_APP_DIR.
2. Faça push na branch main ou execute o workflow "Deploy".
3. Acesse ${PUBLIC_URL_INPUT} e entre com o administrador.

Segurança: use um proxy reverso (Caddy/Nginx) com HTTPS na frente da porta ${APP_PORT}.
EOF
	chown "$APP_USER":"$APP_USER" "$APP_DIR/DEPLOY-INFO.txt"

	if [[ "$env_created" -eq 1 ]]; then
		echo
		log "Setup concluído."
		log "Administrador: ${ADMIN_EMAIL}"
		if [[ "${NO_PRINT_SECRETS:-0}" == "1" ]]; then
			log "Senha inicial salva em: $APP_DIR/shared/INITIAL-ADMIN-PASSWORD.txt"
		else
			log "Senha inicial: ${ADMIN_PASSWORD}"
			warn "Guarde essa senha agora; ela não será exibida novamente."
		fi
	else
		log "Setup concluído. Configurações existentes preservadas em $APP_DIR/shared/.env"
	fi
}

prepare_release() {
	[[ -f "$ARTIFACT" ]] || die "Artefato não encontrado: $ARTIFACT"
	[[ -n "$TAG" ]] || TAG="$(date -u +%Y%m%d%H%M%S)"
	[[ "$TAG" =~ ^[A-Za-z0-9._-]+$ ]] || die "Tag inválida: $TAG"

	local release_dir="$APP_DIR/releases/$TAG"
	if [[ -e "$release_dir" ]]; then
		TAG="${TAG}-$(date -u +%H%M%S)"
		release_dir="$APP_DIR/releases/$TAG"
	fi
	local staging="$APP_DIR/releases/.staging-$TAG-$$"
	rm -rf "$staging"
	mkdir -p "$staging"

	log "Extraindo artefato em $release_dir..."
	tar -xzf "$ARTIFACT" -C "$staging"
	[[ -f "$staging/server" ]] || die "O artefato não contém o binário 'server'."
	[[ -d "$staging/static" ]] || die "O artefato não contém a pasta 'static'."
	chmod +x "$staging/server"
	ln -sfn "$APP_DIR/shared/.env" "$staging/.env"

	mv "$staging" "$release_dir"
	chown -R "$APP_USER":"$APP_USER" "$release_dir"
	echo "$release_dir"
}

activate_release() {
	local release_dir="$1"
	local previous=""
	if [[ -f "$APP_DIR/.current-release" ]]; then
		previous="$(cat "$APP_DIR/.current-release")"
	fi
	if [[ -z "$previous" && -L "$APP_DIR/current" ]]; then
		previous="$(readlink -f "$APP_DIR/current" || true)"
	fi
	if [[ -e "$APP_DIR/current" && ! -L "$APP_DIR/current" ]]; then
		rm -rf "$APP_DIR/current"
	fi

	log "Ativando release $(basename "$release_dir")..."
	ln -sfn "$release_dir" "$APP_DIR/current"
	printf '%s\n' "$release_dir" > "$APP_DIR/.current-release"
	write_ecosystem "$APP_DIR/ecosystem.config.cjs"

	run_as_app pm2 startOrReload "$APP_DIR/ecosystem.config.cjs" --update-env
	run_as_app pm2 save >/dev/null 2>&1 || true
	echo "$previous"
}

wait_health() {
	if [[ "$SKIP_HEALTH" -eq 1 ]]; then
		return 0
	fi
	local url="http://127.0.0.1:${APP_PORT}${HEALTH_PATH}"
	log "Aguardando health check em ${url}..."
	for _ in $(seq 1 60); do
		if curl -fsS --max-time 3 "$url" >/dev/null 2>&1; then
			log "Health check OK."
			return 0
		fi
		sleep 2
	done
	warn "Health check falhou após 120s."
	run_as_app pm2 logs "$APP_NAME" --lines 80 --nostream >&2 || true
	return 1
}

rollback_to() {
	local target="$1"
	[[ -n "$target" && -d "$target" ]] || die "Release anterior não encontrada."
	log "Revertendo para $(basename "$target")..."
	ln -sfn "$target" "$APP_DIR/current"
	printf '%s\n' "$target" > "$APP_DIR/.current-release"
	write_ecosystem "$APP_DIR/ecosystem.config.cjs"
	run_as_app pm2 startOrReload "$APP_DIR/ecosystem.config.cjs" --update-env
	run_as_app pm2 save >/dev/null 2>&1 || true
}

cleanup_releases() {
	local current
	current="$(cat "$APP_DIR/.current-release" 2>/dev/null || true)"
	[[ -n "$current" ]] || current="$(readlink -f "$APP_DIR/current" 2>/dev/null || true)"
	while IFS= read -r dir; do
		[[ -n "$dir" ]] || continue
		dir="${dir%/}"
		[[ "$dir" == "$current" ]] && continue
		rm -rf "$dir"
	done < <(ls -1dt "$APP_DIR"/releases/*/ 2>/dev/null | tail -n +$((KEEP_RELEASES + 1)))
}

deploy() {
	[[ -d "$APP_DIR" ]] || die "Diretório não encontrado: $APP_DIR. Execute o setup primeiro."
	if [[ ! -f "$APP_DIR/shared/.env" ]]; then
		[[ "$(id -u)" -eq 0 ]] || die "Primeiro deploy: execute 'bash deploy.sh setup' como root."
		log "Primeira execução detectada; executando setup automático..."
		NO_PRINT_SECRETS="${NO_PRINT_SECRETS:-1}" setup
	fi
	[[ -f "$APP_DIR/shared/.env" ]] || die "Arquivo $APP_DIR/shared/.env não encontrado. Execute o setup primeiro."
	command -v pm2 >/dev/null 2>&1 || die "PM2 não encontrado. Execute o setup primeiro."
	[[ "$(id -un)" == "$APP_USER" ]] || warn "Executando como $(id -un); o PM2 será controlado como $APP_USER."

	local release_dir previous
	release_dir="$(prepare_release)"
	previous="$(activate_release "$release_dir")"

	if ! wait_health; then
		if [[ -n "$previous" ]]; then
			warn "Rollback automático para a release anterior."
			rollback_to "$previous"
			wait_health || true
		fi
		die "Deploy falhou."
	fi

	cleanup_releases
	if [[ "$KEEP_ARTIFACT" -eq 0 && "$ARTIFACT" != "$APP_DIR"/* ]]; then
		rm -f "$ARTIFACT"
	fi
	log "Deploy concluído: $TAG"
	run_as_app pm2 status "$APP_NAME" || true
}

rollback() {
	[[ -d "$APP_DIR" ]] || die "Diretório não encontrado: $APP_DIR."
	command -v pm2 >/dev/null 2>&1 || die "PM2 não encontrado."
	local current previous
	current="$(cat "$APP_DIR/.current-release" 2>/dev/null || true)"
	[[ -n "$current" ]] || current="$(readlink -f "$APP_DIR/current" 2>/dev/null || true)"
	previous="$(ls -1dt "$APP_DIR"/releases/*/ 2>/dev/null | while IFS= read -r dir; do
		dir="${dir%/}"
		[[ "$dir" == "$current" ]] && continue
		echo "$dir"
		break
	done)"
	[[ -n "$previous" ]] || die "Nenhuma release anterior disponível."
	rollback_to "$previous"
	wait_health
	log "Rollback concluído: $(basename "$previous")"
}

case "$MODE" in
	setup) setup ;;
	rollback) rollback ;;
	deploy)
		if [[ -z "$ARTIFACT" ]]; then
			usage
			die "Informe --artifact (ou use 'setup'/'rollback')."
		fi
		deploy
		;;
esac
