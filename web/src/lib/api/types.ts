/**
 * Wire types of the start-and-found Go API.
 *
 * Collection endpoints answer with {@link Page} and every failure with
 * {@link ApiErrorEnvelope}, so the web app never has to guess a shape.
 */

/** A post is published either by a user or by one of their projects. */
export type PublisherKind = "user" | "project";

/** Lifecycle status of a project. */
export type ProjectStatus = "idea" | "building" | "launched" | "paused" | "archived";

/** Role of a project team member. */
export type MemberRole = "owner" | "admin" | "member";

/** Kind of a post attachment. */
export type MediaKind = "image" | "gif" | "video";

/** Scope of the activity shown on a profile. */
export type ActivityScope = "all" | "user" | "projects";

/** HTTP methods the BFF proxy forwards to the Go API. */
export type ApiMethod = "GET" | "POST" | "PATCH" | "DELETE";

/** The human account that wrote a post, even when a project published it. */
export interface Author {
  id: string;
  username: string;
  display_name: string;
  avatar_url: string;
}

/** A media attachment of a post. */
export interface Media {
  id: string;
  kind: MediaKind;
  mime_type: string;
  url: string;
  width: number;
  height: number;
}

/** A single entry of a thread. */
export interface Post {
  id: string;
  publisher_id: string;
  publisher_handle: string;
  publisher_kind: PublisherKind;
  author: Author;
  parent_id?: string;
  root_id?: string;
  body: string;
  reply_count: number;
  media: Media[];
  created_at: string;
  updated_at: string;
}

/** Public representation of a user. */
export interface Profile {
  id: string;
  username: string;
  handle: string;
  display_name: string;
  avatar_url: string;
  bio: string;
  created_at: string;
}

/** Private representation of a user, returned to its owner only. */
export interface Account extends Profile {
  email: string;
  updated_at: string;
}

/** A public project profile. */
export interface Project {
  id: string;
  owner_id: string;
  owner_username: string;
  handle: string;
  name: string;
  slug: string;
  logo_url: string;
  description: string;
  website: string;
  category: string;
  status: ProjectStatus;
  created_at: string;
  updated_at: string;
}

/** A member of a project team. */
export interface Member {
  user_id: string;
  username: string;
  handle: string;
  display_name: string;
  avatar_url: string;
  role: MemberRole;
  created_at: string;
}

/** A publishing identity: a user or one of their projects. */
export interface Publisher {
  id: string;
  kind: PublisherKind;
  user_id?: string;
  project_id?: string;
  handle: string;
  created_at: string;
}

/** One post of a thread together with its replies. */
export interface ThreadNode {
  post: Post;
  depth: number;
  children: ThreadNode[];
}

/** Payload of GET /threads/{postID}. */
export interface ThreadView {
  root: ThreadNode;
  post_count: number;
  max_depth: number;
}

/** Keyset paginated collection. */
export interface Page<T> {
  items: T[];
  next_cursor?: string;
}

/** Wrapper used by the endpoints that return a plain list. */
export interface ItemsResponse<T> {
  items: T[];
}

/** Payload of GET /users/{username}: profile, showcase and latest activity. */
export interface ProfileView {
  user: Profile;
  projects: Project[];
  posts: Page<Post>;
}

/** Token pair issued by the auth endpoints. */
export interface AuthTokens {
  access_token: string;
  access_expires_at: string;
  refresh_token: string;
  refresh_expires_at: string;
}

/** Payload of POST /auth/signup, /auth/login and /auth/refresh. */
export interface AuthSession extends AuthTokens {
  user: Account;
}

/** Where and how to upload a file after POST /media/presign. */
export interface MediaUpload {
  media_id: string;
  storage_key: string;
  upload_url: string;
  method: string;
  expires_at: string;
  headers: Record<string, string>;
}

/** Body of POST /media/presign. */
export interface PresignInput {
  filename: string;
  content_type: string;
  size_bytes: number;
}

/** Body of POST /posts and POST /posts/{postID}/replies. */
export interface CreatePostInput {
  as: string;
  body: string;
  media_ids: string[];
}

/** Body of POST /auth/signup. */
export interface SignupInput {
  username: string;
  email: string;
  display_name: string;
  password: string;
}

/** Body of POST /auth/login: a username or an email address. */
export interface LoginInput {
  identifier: string;
  password: string;
}

/** Body of PATCH /me. */
export interface UpdateProfileInput {
  display_name?: string;
  avatar_url?: string;
  bio?: string;
}

/** Machine readable failure details, for example { field: "username" }. */
export type ApiErrorDetails = Record<string, unknown>;

/** The error object of the platform envelope. */
export interface ApiErrorBody {
  code: string;
  message: string;
  details?: ApiErrorDetails;
}

/** Envelope used by every failed request. */
export interface ApiErrorEnvelope {
  error: ApiErrorBody;
}

/** Discriminated result helper for callers that prefer values over throws. */
export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: ApiErrorBody & { status: number } };
