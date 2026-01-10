"use client";

import { ApolloProvider as BaseApolloProvider } from "@apollo/client/react";
import { apolloClient } from "@/lib/graphql/client";

interface ApolloProviderProps {
  children: React.ReactNode;
}

/**
 * ApolloProvider wrapper for the application
 * Provides Apollo Client context for GraphQL queries, mutations, and subscriptions
 */
export function ApolloProvider({ children }: ApolloProviderProps) {
  return (
    <BaseApolloProvider client={apolloClient}>
      {children}
    </BaseApolloProvider>
  );
}

export default ApolloProvider;
