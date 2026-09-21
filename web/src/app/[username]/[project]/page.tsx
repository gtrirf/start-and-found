import Link from "next/link";
import { notFound } from "next/navigation";
import Avatar from "@/components/Avatar";
import EmptyState from "@/components/EmptyState";
import PostCard from "@/components/PostCard";
import { ProjectStatusPill } from "@/components/ProjectCard";
import { errorMessage, isApiError } from "@/lib/api/error";
import { apiFetch } from "@/lib/api/server";
import type { ItemsResponse, Member, Page, Post, Project } from "@/lib/api/types";
import { relativeTime } from "@/lib/format";

export const dynamic = "force-dynamic";

interface ProjectPageProps {
  params: Promise<{ username: string; project: string }>;
}

/** Posts requested when collecting the activity of a project publisher. */
const PROJECT_POST_LIMIT = 50;

const MEMBER_ROLE_CLASSES: Record<string, string> = {
  owner: "border-sky-400/30 bg-sky-400/10 text-sky-300",
  admin: "border-amber-400/30 bg-amber-400/10 text-amber-300",
  member: "border-white/15 bg-white/5 text-zinc-400",
};

export default async function ProjectPage({ params }: ProjectPageProps) {
  const { username, project: slug } = await params;
  const projectPath = `/projects/${encodeURIComponent(username)}/${encodeURIComponent(slug)}`;

  let project: Project;
  try {
    project = await apiFetch<Project>(projectPath);
  } catch (error) {
    if (isApiError(error) && error.status === 404) notFound();
    return (
      <EmptyState
        tone="error"
        title="This project is unavailable"
        description={errorMessage(error)}
      />
    );
  }

  let members: Member[] = [];
  try {
    const page = await apiFetch<ItemsResponse<Member>>(`${projectPath}/members`);
    members = Array.isArray(page.items) ? page.items : [];
  } catch {
    members = [];
  }

  // The API exposes project activity through the owner profile; the publisher
  // handle ("@hanzo/sonarai") identifies the posts written for this project.
  let posts: Post[] = [];
  let postsFailure: string | null = null;
  try {
    const page = await apiFetch<Page<Post>>(`/users/${encodeURIComponent(username)}/posts`, {
      query: { scope: "projects", limit: PROJECT_POST_LIMIT },
    });
    const handle = project.handle.toLowerCase();
    posts = page.items.filter((post) => post.publisher_handle.toLowerCase() === handle);
  } catch (error) {
    postsFailure = errorMessage(error);
  }

  return (
    <div className="space-y-6">
      <header className="rounded-xl border border-white/10 bg-white/[0.02] p-5">
        <div className="flex items-start gap-4">
          <Avatar name={project.name} src={project.logo_url} size="lg" />
          <div className="min-w-0 flex-1 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-lg font-semibold text-zinc-50">{project.name}</h1>
              <ProjectStatusPill status={project.status} />
            </div>
            <p className="text-sm text-zinc-500">{project.handle}</p>
            {project.description.length > 0 ? (
              <p className="text-sm text-zinc-400">{project.description}</p>
            ) : null}
            <div className="flex flex-wrap items-center gap-3 pt-1 text-xs text-zinc-500">
              {project.category.length > 0 ? <span>{project.category}</span> : null}
              {project.website.length > 0 ? (
                <a
                  href={project.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sky-400 transition-colors hover:text-sky-300"
                >
                  {project.website}
                </a>
              ) : null}
              <span>Started {relativeTime(project.created_at)}</span>
            </div>
          </div>
        </div>
      </header>

      <section className="space-y-3">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Team</h2>
        {members.length === 0 ? (
          <EmptyState
            title="No team members listed"
            description="Nobody is listed on this project yet."
          />
        ) : (
          <ul className="grid gap-2 sm:grid-cols-2">
            {members.map((member) => {
              const name = member.display_name.length > 0 ? member.display_name : member.username;
              const roleClasses = MEMBER_ROLE_CLASSES[member.role] ?? MEMBER_ROLE_CLASSES.member;
              return (
                <li
                  key={member.user_id}
                  className="flex items-center gap-3 rounded-xl border border-white/10 bg-white/[0.02] p-3"
                >
                  <Avatar name={name} src={member.avatar_url} size="sm" />
                  <div className="min-w-0 flex-1">
                    <Link
                      href={`/${member.username}`}
                      className="block truncate text-sm font-medium text-zinc-100 hover:text-sky-300"
                    >
                      {name}
                    </Link>
                    <p className="truncate text-xs text-zinc-500">{member.handle}</p>
                  </div>
                  <span
                    className={`rounded-full border px-2 py-0.5 text-[11px] font-medium capitalize ${roleClasses}`}
                  >
                    {member.role}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section className="space-y-3">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
          Published by this project
        </h2>
        {postsFailure !== null ? (
          <EmptyState
            tone="error"
            title="Project posts are unavailable"
            description={postsFailure}
          />
        ) : posts.length === 0 ? (
          <EmptyState
            title="Nothing published yet"
            description={`${project.name} has no posts yet.`}
          />
        ) : (
          <ul className="space-y-3">
            {posts.map((post) => (
              <li key={post.id}>
                <PostCard post={post} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
