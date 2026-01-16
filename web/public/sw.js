// AgentsMesh Service Worker
const CACHE_NAME = 'agentsmesh-v1';
const STATIC_CACHE_NAME = 'agentsmesh-static-v1';

// Static assets to cache on install
const STATIC_ASSETS = [
  '/offline',
  '/manifest.json',
  '/icons/icon.svg',
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE_NAME).then((cache) => {
      return cache.addAll(STATIC_ASSETS);
    })
  );
  self.skipWaiting();
});

// Activate event - clean old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((name) => name !== CACHE_NAME && name !== STATIC_CACHE_NAME)
          .map((name) => caches.delete(name))
      );
    })
  );
  self.clients.claim();
});

// Fetch event - network first with cache fallback
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip non-GET requests
  if (request.method !== 'GET') return;

  // Skip WebSocket connections
  if (url.protocol === 'ws:' || url.protocol === 'wss:') return;

  // Skip API requests (don't cache)
  if (url.pathname.startsWith('/api/')) {
    return;
  }

  // Skip chrome-extension and other non-http(s) requests
  if (!url.protocol.startsWith('http')) return;

  event.respondWith(
    fetch(request)
      .then((response) => {
        // Clone the response for caching
        const responseClone = response.clone();

        // Cache successful responses for static assets
        if (response.status === 200) {
          const isStatic =
            url.pathname.startsWith('/_next/static/') ||
            url.pathname.match(/\.(js|css|png|jpg|jpeg|gif|svg|ico|woff|woff2)$/);

          if (isStatic) {
            caches.open(STATIC_CACHE_NAME).then((cache) => {
              cache.put(request, responseClone);
            });
          }
        }

        return response;
      })
      .catch(() => {
        // Return cached response if available
        return caches.match(request).then((cachedResponse) => {
          if (cachedResponse) {
            return cachedResponse;
          }

          // Return offline page for navigation requests
          if (request.mode === 'navigate') {
            return caches.match('/offline');
          }

          return new Response('Offline', {
            status: 503,
            statusText: 'Service Unavailable',
          });
        });
      })
  );
});

// Push notification event
self.addEventListener('push', (event) => {
  if (!event.data) return;

  const data = event.data.json();
  const { title, body, icon, tag, data: notificationData } = data;

  const options = {
    body,
    icon: icon || '/icons/icon.svg',
    badge: '/icons/icon.svg',
    tag: tag || 'agentsmesh-notification',
    data: notificationData,
    vibrate: [100, 50, 100],
    actions: notificationData?.actions || [],
  };

  event.waitUntil(
    self.registration.showNotification(title || 'AgentsMesh', options)
  );
});

// Notification click event
self.addEventListener('notificationclick', (event) => {
  event.notification.close();

  const notificationData = event.notification.data;
  let targetUrl = '/';

  if (notificationData) {
    switch (notificationData.type) {
      case 'pod_status':
        targetUrl = `/${notificationData.orgSlug}/workspace`;
        break;
      case 'ticket_assigned':
      case 'ticket_updated':
        targetUrl = `/${notificationData.orgSlug}/tickets/${notificationData.ticketIdentifier}`;
        break;
      case 'runner_offline':
        targetUrl = `/${notificationData.orgSlug}/runners`;
        break;
      default:
        targetUrl = notificationData.url || '/';
    }
  }

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
      // Focus existing window if available
      for (const client of windowClients) {
        if (client.url.includes(targetUrl) && 'focus' in client) {
          return client.focus();
        }
      }
      // Open new window
      if (clients.openWindow) {
        return clients.openWindow(targetUrl);
      }
    })
  );
});

// Background sync for offline actions
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-actions') {
    event.waitUntil(syncOfflineActions());
  }
});

async function syncOfflineActions() {
  // Sync any queued offline actions when back online
  // This could be implemented to queue terminal commands, etc.
  console.log('[SW] Syncing offline actions...');
}

// Message handler for communication with main thread
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
