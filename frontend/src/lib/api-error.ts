import { GITHUB_TOKEN_DOCS_URL } from '@/lib/constants';

interface APIError {
  apiErrorType?: string;
  apiErrorSource?: string;
  apiStatusCode?: number;
}

function isGitHubSource(source: string | undefined): boolean {
  return (source ?? '').toLowerCase() === 'github';
}

export function apiErrorMessage(error: APIError): string | null {
  if (!isGitHubSource(error.apiErrorSource)) {
    // TODO: If we add more API sources, we'll want to handle them here as well
    return null;
  }

  const statusMessage =
    typeof error.apiStatusCode === 'number' && error.apiStatusCode > 0
      ? ` with status ${error.apiStatusCode}`
      : '';

  switch (error.apiErrorType) {
    case 'api_auth_invalid_token':
      return `GitHub API token is invalid or unauthorized. Add or update your token: ${GITHUB_TOKEN_DOCS_URL}`;
    case 'api_rate_limited':
      return `GitHub API rate limit reached. Add a GitHub token to increase limits: ${GITHUB_TOKEN_DOCS_URL}`;
    case 'api_forbidden':
      return `GitHub API request was forbidden. Check your token scopes/permissions: ${GITHUB_TOKEN_DOCS_URL}`;
    case 'api_bad_request':
      return `GitHub API rejected the request${statusMessage}. Please retry or report this issue if it persists.`;
    case 'api_not_found':
      return `GitHub resource was not found${statusMessage}. The repository or release may no longer exist.`;
    case 'api_upstream_5xx':
      return `GitHub API is currently unavailable${statusMessage}. Please try again once GitHub is back online.`;
    case 'api_timeout':
      return `GitHub API request timed out. Please check your network connection and retry.`;
    case 'api_network_error':
      return `Network error while contacting GitHub API. Please check your network connection and retry.`;
    case 'api_fetch_error':
      return `Failed to fetch GitHub API data${statusMessage}. Please check your network connection and retry.`;
    case 'api_status_error':
      return `GitHub API request failed${statusMessage}. Please retry.`;
    default:
      return null;
  }
}

export function apiErrorMessages(
  errors: unknown[] | undefined | null,
): string[] {
  const messages: string[] = [];
  const seen = new Set<string>();

  for (const error of errors ?? []) {
    const candidate = toAPIError(error);
    const message = apiErrorMessage(candidate);
    if (!message || seen.has(message)) {
      continue;
    }
    seen.add(message);
    messages.push(message);
  }
  return messages;
}

function toAPIError(value: unknown): APIError {
  if (typeof value !== 'object' || value === null) {
    return {};
  }

  const record = value as Record<string, unknown>;
  return {
    apiErrorType:
      typeof record.apiErrorType === 'string' ? record.apiErrorType : undefined,
    apiErrorSource:
      typeof record.apiErrorSource === 'string'
        ? record.apiErrorSource
        : undefined,
    apiStatusCode:
      typeof record.apiStatusCode === 'number'
        ? record.apiStatusCode
        : undefined,
  };
}
