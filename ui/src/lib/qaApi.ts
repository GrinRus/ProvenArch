import { fetchJSON } from "./api";

export type QAAskCitation = {
  path: string;
  reason: string;
};

export type QAAskResponse = {
  answer: string;
  citations: QAAskCitation[];
  unresolved: string[];
  confidence: number;
};

type QAAskRawResponse = {
  answer?: string | null;
  citations?: QAAskCitation[] | null;
  unresolved?: string[] | null;
  confidence?: number | null;
};

export async function askArchitectureQuestion(question: string): Promise<QAAskResponse> {
  const response = await fetchJSON<QAAskRawResponse>("/api/qa/ask", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question }),
  });
  return {
    answer: response.answer ?? "",
    citations: Array.isArray(response.citations) ? response.citations : [],
    unresolved: Array.isArray(response.unresolved) ? response.unresolved : [],
    confidence: typeof response.confidence === "number" ? response.confidence : 0,
  };
}
