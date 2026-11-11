import { useEffect, useState } from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import {
  Asleep,
  Calendar,
  ChartBar,
  Checkmark,
  Close,
  Document,
  Home,
  Light,
  List,
  Link,
  Logout,
  Menu,
  Notification,
  Settings,
  SignalStrength,
  UserMultiple,
  UserProfile,
} from '@carbon/icons-react';
import {
  Header,
  HeaderGlobalAction,
  HeaderGlobalBar,
  HeaderMenuButton,
  HeaderName,
  OverflowMenu,
  OverflowMenuItem,
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
  Theme,
} from '@carbon/react';
import { useAuth } from '../contexts/AuthContext';
import { useAppTheme } from '../contexts/ThemeContext';
import type { Module } from '../lib/types';

function iconFor(key: string) {
  switch (key) {
    case 'dashboard':
      return Home;
    case 'users':
      return UserMultiple;
    case 'dev-docs':
      return Document;
    case 'agenda':
      return Calendar;
    case 'tasks':
      return Checkmark;
    case 'follow-ups':
      return List;
    case 'stations':
    case 'nokia':
      return SignalStrength;
    case 'links':
      return Link;
    default:
      return Home;
  }
}

function useIsMobile(): boolean {
  const [mobile, setMobile] = useState(() => window.matchMedia('(max-width: 1055px)').matches);
  useEffect(() => {
    const media = window.matchMedia('(max-width: 1055px)');
    const listener = (event: MediaQueryListEvent) => setMobile(event.matches);
    media.addEventListener('change', listener);
    return () => media.removeEventListener('change', listener);
  }, []);
  return mobile;
}

function NavItem({
  module,
  active,
}: {
  module: Module;
  active: boolean;
}) {
  const Icon = iconFor(module.key);
  return (
    <SideNavLink
      as={NavLink}
      to={module.path}
      isActive={active}
      renderIcon={Icon}
    >
      {module.label}
    </SideNavLink>
  );
}

export function AppLayout({ children }: { children: React.ReactNode }) {
  const { user, modules, unread, logout } = useAuth();
  const { preference, cyclePreference } = useAppTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const isMobile = useIsMobile();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  if (!user) return null;

  const sidebarModules = modules.filter((module) => !module.system_link && module.section === 'sidebar');
  const groupedModules = modules.filter((module) => !module.system_link && module.section === 'modules');
  const systemLinks = modules.filter((module) => module.system_link);
  const themeIcon = preference === 'dark' ? Asleep : Light;
  const ThemeIcon = themeIcon;

  const go = (path: string) => navigate(path);

  return (
    <div className="app-shell">
      <Theme theme="g100">
        <Header aria-label="Documentation Center" className="app-header">
          <HeaderMenuButton
            aria-label={mobileOpen ? 'Fechar menu' : 'Abrir menu'}
            isCollapsible
            isActive={mobileOpen}
            onClick={() => setMobileOpen((open) => !open)}
            className="app-header__menu"
            renderMenuIcon={<Menu size={20} />}
            renderCloseIcon={<Close size={20} />}
          />
          <HeaderName href="/dashboard" prefix="" onClick={(event) => { event.preventDefault(); go('/dashboard'); }}>
            Documentation Center
          </HeaderName>
          <HeaderGlobalBar>
            <HeaderGlobalAction
              aria-label={unread > 0 ? `Notificações (${unread} não lidas)` : 'Notificações'}
              onClick={() => go('/notifications')}
              className="header-notification"
            >
              <Notification size={20} />
              {unread > 0 && <span className="header-notification__badge">{unread > 9 ? '9+' : unread}</span>}
            </HeaderGlobalAction>
            <HeaderGlobalAction
              aria-label={`Tema: ${preference}. Alternar tema`}
              onClick={() => void cyclePreference()}
              tooltipAlignment="end"
            >
              <ThemeIcon size={20} />
            </HeaderGlobalAction>
            <OverflowMenu
              aria-label="Menu do usuário"
              iconDescription="Menu do usuário"
              renderIcon={UserProfile}
              direction="bottom"
              className="user-menu"
            >
              <div className="user-menu__identity">
                <strong>{user.full_name}</strong>
                <span>{user.email}</span>
              </div>
              <OverflowMenuItem itemText="Perfil" onClick={() => go('/profile')} />
              <OverflowMenuItem itemText="Configurações" onClick={() => go('/settings')} />
              <OverflowMenuItem
                itemText="Notificações"
                onClick={() => go('/notifications')}
              />
              <OverflowMenuItem
                itemText="Sair"
                hasDivider
                isDelete
                onClick={() => {
                  void logout().then(() => navigate('/login'));
                }}
              />
            </OverflowMenu>
          </HeaderGlobalBar>
        </Header>
      </Theme>

      {isMobile && mobileOpen && (
        <button
          type="button"
          className="sidebar-overlay"
          aria-label="Fechar navegação"
          onClick={() => setMobileOpen(false)}
        />
      )}

      <SideNav
        aria-label="Navegação principal"
        expanded={!isMobile || mobileOpen}
        isChildOfHeader
        className="app-sidebar"
        onOverlayClick={() => setMobileOpen(false)}
      >
        <SideNavItems>
          {sidebarModules.map((module) => (
            <NavItem
              key={module.key}
              module={module}
              active={location.pathname === module.path || location.pathname.startsWith(`${module.path}/`)}
            />
          ))}
          {groupedModules.length > 0 && (
            <SideNavMenu title="Módulos" defaultExpanded>
              {groupedModules.map((module) => {
                return (
                  <SideNavMenuItem
                    key={module.key}
                    as={NavLink}
                    to={module.path}
                    isActive={location.pathname === module.path || location.pathname.startsWith(`${module.path}/`)}
                  >
                    {module.label}
                  </SideNavMenuItem>
                );
              })}
            </SideNavMenu>
          )}
          <SideNavLink
            as={NavLink}
            to="/notifications"
            renderIcon={Notification}
            isActive={location.pathname.startsWith('/notifications')}
          >
            Notificações
          </SideNavLink>
          <SideNavLink
            as={NavLink}
            to="/settings"
            renderIcon={Settings}
            isActive={location.pathname.startsWith('/settings')}
          >
            Configurações
          </SideNavLink>
          {systemLinks.map((module) => (
            <NavItem
              key={module.key}
              module={module}
              active={location.pathname === module.path || location.pathname.startsWith(`${module.path}/`)}
            />
          ))}
          <SideNavLink
            as={NavLink}
            to="/profile"
            renderIcon={UserProfile}
            isActive={location.pathname.startsWith('/profile')}
          >
            Perfil
          </SideNavLink>
          <SideNavLink
            href="/login"
            renderIcon={Logout}
            onClick={(event) => {
              event.preventDefault();
              void logout().then(() => navigate('/login'));
            }}
          >
            Sair
          </SideNavLink>
        </SideNavItems>
      </SideNav>

      <main id="main-content" className="app-main">
        <div className="app-main__inner">{children}</div>
        <footer className="app-footer">
          <span>Documentation Center</span>
          <span>Go + React + IBM Carbon</span>
        </footer>
      </main>
    </div>
  );
}
