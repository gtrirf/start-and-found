import Link from "next/link";
import Avatar from "./Avatar";
import type { Project, ProjectStatus } from "@/lib/api/types";

export interface ProjectCardProps {
  project: Project;
  className?: string;
}

const STATUS_CLASSES: Record<ProjectStatus, string> = {
  idea: "border-zinc-400/30 bg-zinc-400/10 text-zinc-300",
  building: "border-amber-400/30 bg-amber-400/10 text-amber-300",
  launched: "border-emerald-400/30 bg-emerald-400/10 text-emerald-300",
  paused: "border-orange-400/30 bg-orange-400/10 text-orange-300",
  archived: "border-white/15 bg-white/5 text-zinc-400",
};

/** Coloured pill rendering the lifecycle status of a project. */
export function ProjectStatusPill({
  status,
  className = "",
}: {
  status: ProjectStatus;
  className?: string;
}) {
  const styles = STATUS_CLASSES[status] ?? STATUS_CLASSES.idea;
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium capitalize ${styles} ${className}`.trim()}
    >
      {status}
    </span>
  );
}

/** Showcase card of a project on a profile page. */
export default function ProjectCard({ project, className = "" }: ProjectCardProps) {
  return (
    <Link
      href={`/${project.owner_username}/${project.slug}`}
      className={`group flex flex-col gap-3 rounded-xl border border-white/10 bg-white/[0.02] p-4 transition-colors hover:border-sky-400/40 ${className}`.trim()}
    >
      <div className="flex items-start gap-3">
        <Avatar name={project.name} src={project.logo_url} size="md" />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate text-sm font-semibold text-zinc-100 group-hover:text-sky-300">
              {project.name}
            </h3>
            <ProjectStatusPill status={project.status} />
          </div>
          <p className="truncate text-xs text-zinc-500">{project.handle}</p>
        </div>
      </div>

      {project.description.length > 0 ? (
        <p className="line-clamp-3 text-sm text-zinc-400">{project.description}</p>
      ) : null}

      {project.category.length > 0 ? (
        <p className="text-xs text-zinc-600">{project.category}</p>
      ) : null}
    </Link>
  );
}
