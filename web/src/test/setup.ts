import '@testing-library/jest-dom'
import { vi, afterEach } from 'vitest'

// Mock localStorage for Zustand persist middleware
const createLocalStorageMock = () => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
    get length() {
      return Object.keys(store).length;
    },
    key: (index: number) => {
      const keys = Object.keys(store);
      return keys[index] || null;
    },
  };
};

Object.defineProperty(window, 'localStorage', {
  value: createLocalStorageMock(),
  writable: true,
});

Object.defineProperty(window, 'sessionStorage', {
  value: createLocalStorageMock(),
  writable: true,
});

// Mock ResizeObserver
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

global.ResizeObserver = ResizeObserverMock

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock IntersectionObserver
class IntersectionObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

global.IntersectionObserver = IntersectionObserverMock as unknown as typeof IntersectionObserver

// Mock Element.scrollIntoView (not available in jsdom)
Element.prototype.scrollIntoView = vi.fn()

// Clean up after each test
afterEach(() => {
  vi.clearAllMocks()
})
