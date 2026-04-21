import { useParams } from "react-router-dom";
import { LoopDetailPane } from "@/components/loops/LoopDetailPane";

export function LoopDetailPage() {
  const params = useParams<{ org: string; slug: string }>();
  const slug = params.slug ?? "";
  const orgSlug = params.org ?? "";
  return <LoopDetailPane slug={slug} orgSlug={orgSlug} />;
}
