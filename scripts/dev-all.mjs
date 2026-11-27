import { spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const isWindows = process.platform === 'win32';

const services = [
  { name: 'backend', args: ['run', 'backend'] },
  { name: 'frontend', args: ['--prefix', 'apps/frontend', 'run', 'dev'] },
];

const children = [];
let stopping = false;

function npmInvocation(args) {
  if (process.env.npm_execpath) {
    return { command: process.execPath, args: [process.env.npm_execpath, ...args] };
  }
  if (isWindows) {
    return { command: 'cmd.exe', args: ['/c', ['npm', ...args].join(' ')] };
  }
  return { command: 'npm', args };
}

function createLogger(name, stream) {
  let buffer = '';

  function write(chunk) {
    buffer += chunk.toString('utf8');
    const lines = buffer.split(/\r?\n/);
    buffer = lines.pop() ?? '';
    for (const line of lines) stream.write(`[${name}] ${line}\n`);
  }

  function flush() {
    if (buffer) {
      stream.write(`[${name}] ${buffer}\n`);
      buffer = '';
    }
  }

  return { write, flush };
}

function stopChild(child) {
  if (child.exitCode !== null || child.signalCode !== null) return;

  if (isWindows) {
    const killer = spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'], {
      stdio: 'ignore',
      windowsHide: true,
    });
    killer.on('error', () => child.kill());
    return;
  }

  try {
    process.kill(-child.pid, 'SIGTERM');
  } catch {
    child.kill('SIGTERM');
  }
}

function stopAll(exitCode) {
  if (stopping) return;
  stopping = true;

  for (const child of children) stopChild(child);
  for (const child of children) {
    if (child.stdout) child.stdout.destroy();
    if (child.stderr) child.stderr.destroy();
  }

  setTimeout(() => process.exit(exitCode), 1200);
}

for (const service of services) {
  const npm = npmInvocation(service.args);
  const child = spawn(npm.command, npm.args, {
    cwd: root,
    shell: false,
    detached: !isWindows,
    windowsHide: true,
  });

  const out = createLogger(service.name, process.stdout);
  const err = createLogger(service.name, process.stderr);

  child.stdout.on('data', out.write);
  child.stderr.on('data', err.write);
  child.on('error', (error) => err.write(`falha ao iniciar: ${error.message}\n`));
  child.on('exit', (code, signal) => {
    out.flush();
    err.flush();
    if (stopping) return;
    process.stderr.write(
      `[dev] ${service.name} encerrado (${signal ?? code}); encerrando os demais.\n`,
    );
    stopAll(code ?? 0);
  });

  children.push(child);
}

process.on('SIGINT', () => stopAll(130));
process.on('SIGTERM', () => stopAll(143));
