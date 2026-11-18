export type ThemePreference = 'system' | 'light' | 'dark';

export interface User {
  id: number;
  name: string;
  surname?: string | null;
  full_name: string;
  email: string;
  is_admin: boolean;
  is_active: boolean;
  theme: ThemePreference;
  created_at: string;
}

export interface Module {
  key: string;
  label: string;
  icon: string;
  path: string;
  section: 'sidebar' | 'modules';
  default_section: 'sidebar' | 'modules';
  toggleable: boolean;
  admin_only: boolean;
  primary: boolean;
  system_link?: boolean;
  enabled: boolean;
}

export interface PageMeta {
  current_page: number;
  last_page: number;
  per_page: number;
  total: number;
  from: number;
  to: number;
}

export interface RecentUser {
  id: number;
  full_name: string;
  email: string;
  is_admin: boolean;
  is_active: boolean;
  created_at: string;
}

export interface DashboardWidget {
  id: string;
  title: string;
  description: string;
  icon: string;
  default_span: number;
  admin_only: boolean;
  data?: {
    total?: number;
    users?: RecentUser[];
    [key: string]: unknown;
  };
}

export interface DashboardResponse {
  greeting: string;
  date: string;
  widgets: DashboardWidget[];
  hidden: string[];
  order: string[];
  spans: Record<string, number>;
  editing: boolean;
  server_time: string;
  default_order: string[];
}

export interface UserStats {
  total: number;
  admins: number;
  active: number;
  inactive: number;
}

export interface UsersResponse {
  data: User[];
  stats: UserStats;
  meta: PageMeta;
  filters: {
    search: string;
    profile: string;
    status: string;
    sort: string;
    direction: string;
    per_page: number;
  };
}

export interface DocSummary {
  id: number;
  title: string;
  slug: string;
  category: string;
  summary: string | null;
  is_published: boolean;
  updated_at: string;
  author_name: string | null;
}

export interface DevDoc extends DocSummary {
  user_id: number;
  content: string;
  created_at: string;
  author_id?: number | null;
}

export interface Category {
  key: string;
  label: string;
}

export interface NavigationGroup {
  category: string;
  label: string;
  items: DocSummary[];
}

export interface DocsResponse {
  data: DocSummary[];
  categories: Category[];
  navigation: NavigationGroup[];
  stats: { total: number; published: number; drafts: number; categories: number };
  meta: PageMeta;
  filters: { search: string; category: string };
  can_manage: boolean;
}

export interface DocShowResponse {
  document: DevDoc;
  html: string;
  category: Category;
  navigation: NavigationGroup[];
  can_manage: boolean;
}

export interface IepmsUser {
  id: number;
  resource_id: string;
  name: string;
  gender: string | null;
  nationality: string | null;
  certificate_type: string | null;
  id_number: string | null;
  birth_year: string | null;
  tel: string | null;
  personal_mailbox: string | null;
  outsourcing_account: string | null;
  last_updated_by: string | null;
  status: 'ativo' | 'inativado';
  has_password: boolean;
  created_at: string;
  updated_at: string;
}

export interface IepmsUsersResponse {
  data: IepmsUser[];
  stats: { total: number; active: number; inactive: number };
  statuses: string[];
  meta: PageMeta;
}

export type FormFieldType =
  | 'text'
  | 'textarea'
  | 'number'
  | 'date'
  | 'time'
  | 'radio'
  | 'checkbox'
  | 'dropdown'
  | 'scale'
  | 'image'
  | 'file';

export type FormFieldStatus =
  | 'pendente'
  | 'enviado'
  | 'aprovado'
  | 'reprovado'
  | 'nao_aplicavel';

export interface FormField {
  id: string;
  type: FormFieldType;
  label: string;
  help?: string;
  required?: boolean;
  options?: string[];
  scaleMin?: number;
  scaleMax?: number;
  scaleMinLabel?: string;
  scaleMaxLabel?: string;
  value?: string;
  images?: string[];
  imageThumbnails?: string[];
  imageDecisions?: Record<string, string>;
  files?: string[];
  fileNames?: string[];
  status?: FormFieldStatus;
}

export interface FormStep {
  id: string;
  title: string;
  fields: FormField[];
}

export interface FormRecord {
  id: number;
  project_id: number | null;
  project_code: string | null;
  project_name: string | null;
  station_id: number | null;
  station_site_id: string | null;
  station_code: string | null;
  station_municipio: string | null;
  station_estado: string | null;
  radio_link_id: number | null;
  radio_link_name: string | null;
  radio_link_code: string | null;
  code: string;
  name: string;
  type: string;
  status: string;
  allows_multiple_responses: boolean;
  public_token: string;
  steps: FormStep[];
  fields: unknown;
  created_at: string;
  updated_at: string;
  response_count: number;
  evaluation_count: number;
}

export interface FormResponseRecord {
  id: number;
  form_id: number;
  user_id: number | null;
  user_name: string | null;
  status: string;
  steps: FormStep[];
  submitted_at: string | null;
  created_at: string;
  updated_at: string;
  evaluations_count: number;
}

export interface FormChange {
  type: 'field' | 'image';
  id: string;
  label: string;
  index?: number;
  from: string | null;
  to: string;
}

export interface FormEvaluationRecord {
  id: number;
  form_id: number;
  form_response_id: number | null;
  user_id: number;
  user_name: string | null;
  status: string;
  notes: string | null;
  changes: FormChange[] | null;
  created_at: string;
  updated_at: string;
}

export interface FormTemplateRecord {
  id: number;
  name: string;
  type: string;
  steps: FormStep[];
  created_at: string;
  updated_at: string;
}

export interface FormSettings {
  types: string[];
  statuses: string[];
  default_template_id: number | null;
  default_view: 'table' | 'cards' | 'compact';
  per_page: number;
}

export interface FormsStats {
  total: number;
  pending: number;
  in_progress: number;
  filled: number;
  approved: number;
  rejected: number;
}

export interface FormsListResponse {
  data: FormRecord[];
  stats: FormsStats;
  statuses: string[];
  types: string[];
  templates: FormTemplateRecord[];
  settings: FormSettings;
  meta: PageMeta;
}

export interface FormDetailResponse {
  data: {
    form: FormRecord;
    responses: FormResponseRecord[];
    evaluations: FormEvaluationRecord[];
    templates: FormTemplateRecord[];
    settings: FormSettings;
  };
}

export interface FormSettingsResponse {
  data: {
    settings: FormSettings;
    templates: FormTemplateRecord[];
    views: Record<string, string>;
    per_page_options: number[];
  };
}

export type NotificationLevel = 'info' | 'success' | 'warning' | 'danger';

export interface AppNotification {
  id: string;
  title: string;
  body: string;
  url: string | null;
  level: NotificationLevel;
  read_at: string | null;
  created_at: string;
}

export interface NotificationsResponse {
  data: AppNotification[];
  latest: AppNotification[];
  unread: number;
  meta: PageMeta;
}

export interface StorageSettings {
  provider: 'local' | 'aws' | 'cloudflare';
  max_upload_mb: number;
  aws_region: string;
  aws_bucket: string;
  aws_endpoint: string;
  aws_public_url: string;
  aws_use_path_style: boolean;
  aws_access_key_set: boolean;
  aws_secret_key_set: boolean;
  cloudflare_account_id: string;
  cloudflare_bucket: string;
  cloudflare_public_url: string;
  cloudflare_access_key_set: boolean;
  cloudflare_secret_key_set: boolean;
  configured: boolean;
}

export interface SettingsResponse {
  user: User;
  modules: Module[];
  enabled_modules: Module[];
  theme: ThemePreference;
  email_notifications: boolean;
  timezone: string;
  timezones: string[];
  is_admin: boolean;
  storage?: StorageSettings;
}

export interface RadioLink {
  id: number;
  name: string;
  code: string | null;
  station_a_id: number;
  station_b_id: number;
  status: string;
  technology: string | null;
  frequency_mhz: number | null;
  capacity_mbps: number | null;
  distance_km: number | null;
  notes: string | null;
  station_a_site_id: string | null;
  station_a_station_id: string | null;
  station_a_municipio: string | null;
  station_a_estado: string | null;
  station_a_tipo_de_elemento: string | null;
  station_b_site_id: string | null;
  station_b_station_id: string | null;
  station_b_municipio: string | null;
  station_b_estado: string | null;
  station_b_tipo_de_elemento: string | null;
  created_at: string;
  updated_at: string;
  attachments: RadioLinkAttachment[];
  comments: RadioLinkComment[];
}

export interface RadioLinkAttachment {
  id: number;
  uploaded_by: number | null;
  original_name: string;
  path: string;
  disk: string;
  mime_type: string | null;
  size: number;
  created_at: string;
  updated_at: string;
  uploader_name: string | null;
  url: string;
}

export interface RadioLinkComment {
  id: number;
  user_id: number | null;
  body: string;
  created_at: string;
  updated_at: string;
  author_name: string | null;
}
