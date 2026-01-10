import {
  ApolloClient,
  InMemoryCache,
  createHttpLink,
  ApolloLink,
} from "@apollo/client";
import { setContext } from "@apollo/client/link/context";
import { ErrorLink } from "@apollo/client/link/error";
import { CombinedGraphQLErrors } from "@apollo/client/errors";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// Create HTTP link for queries and mutations
const httpLink = createHttpLink({
  uri: `${API_URL}/graphql`,
});

// Auth link to add JWT token to requests
const authLink = setContext((_, { headers }) => {
  // Get token from localStorage (client-side only)
  const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;
  const orgSlug = typeof window !== "undefined" ? localStorage.getItem("currentOrg") : null;

  return {
    headers: {
      ...headers,
      authorization: token ? `Bearer ${token}` : "",
      "X-Organization-Slug": orgSlug || "",
    },
  };
});

// Error handling link using Apollo Client v4 API
const errorLink = new ErrorLink(({ error }) => {
  if (CombinedGraphQLErrors.is(error)) {
    error.errors.forEach(({ message, locations, path, extensions }) => {
      console.error(
        `[GraphQL error]: Message: ${message}, Location: ${JSON.stringify(locations)}, Path: ${path}`
      );

      // Handle authentication errors
      if (extensions?.["code"] === "UNAUTHENTICATED") {
        // Clear token and redirect to login
        if (typeof window !== "undefined") {
          localStorage.removeItem("token");
          window.location.href = "/login";
        }
      }
    });
  } else {
    // Network or other errors
    console.error(`[Network error]: ${error}`);
  }
});

// Create Apollo Client instance with HTTP link (no WebSocket subscriptions)
export const apolloClient = new ApolloClient({
  link: ApolloLink.from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache({
    typePolicies: {
      Query: {
        fields: {
          // Merge paginated ticket results
          tickets: {
            keyArgs: ["filter"],
            merge(existing, incoming, { args }) {
              if (!args?.filter?.offset || args.filter.offset === 0) {
                return incoming;
              }
              return {
                ...incoming,
                tickets: [...(existing?.tickets || []), ...incoming.tickets],
              };
            },
          },
          // Merge paginated pod results
          pods: {
            keyArgs: ["filter"],
            merge(existing, incoming, { args }) {
              if (!args?.filter?.offset || args.filter.offset === 0) {
                return incoming;
              }
              return {
                ...incoming,
                pods: [...(existing?.pods || []), ...incoming.pods],
              };
            },
          },
          // Merge paginated channel messages
          channelMessages: {
            keyArgs: ["channelId"],
            merge(existing, incoming, { args }) {
              if (!args?.offset || args.offset === 0) {
                return incoming;
              }
              return {
                ...incoming,
                messages: [...(existing?.messages || []), ...incoming.messages],
              };
            },
          },
        },
      },
      // Normalize entities by ID
      User: {
        keyFields: ["id"],
      },
      Organization: {
        keyFields: ["id"],
      },
      Runner: {
        keyFields: ["id"],
      },
      Pod: {
        keyFields: ["podKey"],
      },
      Channel: {
        keyFields: ["id"],
      },
      Ticket: {
        keyFields: ["identifier"],
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: "cache-and-network",
      errorPolicy: "all",
    },
    query: {
      fetchPolicy: "network-only",
      errorPolicy: "all",
    },
    mutate: {
      errorPolicy: "all",
    },
  },
});

// Helper to reset Apollo cache (useful after logout)
export const resetApolloCache = async () => {
  await apolloClient.clearStore();
};

// Helper to refetch active queries
export const refetchQueries = async (queryNames: string[]) => {
  await apolloClient.refetchQueries({
    include: queryNames,
  });
};
