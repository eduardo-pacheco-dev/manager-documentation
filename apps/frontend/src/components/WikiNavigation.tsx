import { useEffect, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ChevronDown, ChevronRight, Document } from '@carbon/icons-react';
import { Tag } from '@carbon/react';
import type { NavigationGroup } from '../lib/types';

export function WikiNavigation({ groups }: { groups: NavigationGroup[] }) {
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  return (
    <aside className={`wiki-nav ${mobileOpen ? 'wiki-nav--open' : ''}`}>
      <button
        type="button"
        className="wiki-nav__mobile-toggle"
        onClick={() => setMobileOpen((open) => !open)}
        aria-expanded={mobileOpen}
      >
        <Document size={16} />
        Navegação da wiki
        {mobileOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </button>
      <div className="wiki-nav__body">
        <p className="wiki-nav__title">Documentação</p>
        {groups.map((group) => (
          <div key={group.category} className="wiki-nav__group">
            <p className="wiki-nav__group-label">{group.label}</p>
            <ul>
              {group.items.map((item) => {
                const active = location.pathname === `/dev-docs/${item.id}`;
                return (
                  <li key={item.id}>
                    <Link to={`/dev-docs/${item.id}`} className={active ? 'is-active' : ''}>
                      <span>{item.title}</span>
                      {!item.is_published && <Tag type="cool-gray" size="sm">Rascunho</Tag>}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </div>
    </aside>
  );
}
