/**
 * Test Configuration for Dual-Mode E2E Testing
 * Supports both Simple-Mode and SaaS-Mode
 */

const TEST_MODES = {
  SIMPLE: 'simple',
  SAAS: 'saas'
};

/**
 * Get test configuration based on mode
 * Mode is determined by PLAYWRIGHT_TEST_MODE env var or project name
 */
function getTestConfig(mode = process.env.PLAYWRIGHT_TEST_MODE || TEST_MODES.SAAS) {
  const configs = {
    // Simple-Mode: Now uses demo tenant in SaaS mode (same as SAAS mode)
    // Server runs in SaaS mode, so we use demo tenant for all tests
    [TEST_MODES.SIMPLE]: {
      mode: TEST_MODES.SIMPLE,
      // Use demo tenant URL since server is in SaaS mode
      baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',

      // Demo tenant credentials (same as SaaS mode)
      credentials: {
        admin: {
          email: 'admin@demo.gassigeher.local',
          password: process.env.DEMO_ADMIN_PASSWORD || 'demo1234'
        },
        // Demo users have fixed password
        greenUser: {
          email: 'anna@demo.gassigeher.local',
          password: 'demo1234'
        },
        orangeUser: {
          email: 'bernd@demo.gassigeher.local',
          password: 'demo1234'
        },
        blueUser: {
          email: 'clara@demo.gassigeher.local',
          password: 'demo1234'
        }
      },

      // Page paths are at root (served from tenant subdomain)
      paths: {
        login: '/login.html',
        register: '/register.html',
        dashboard: '/dashboard.html',
        dogs: '/dogs.html',
        profile: '/profile.html',
        terms: '/terms.html',
        privacy: '/privacy.html',
        forgotPassword: '/forgot-password.html',
        adminDashboard: '/admin-dashboard.html'
      }
    },

    // SaaS-Mode: Multi-tenant, access via tenant subdomain
    [TEST_MODES.SAAS]: {
      mode: TEST_MODES.SAAS,
      // Use demo tenant subdomain
      baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
      // Main domain for landing/central admin
      mainDomainURL: process.env.SAAS_MAIN_URL || 'http://gassigeher.local:8080',

      // Demo tenant credentials (all demo users use same password)
      credentials: {
        admin: {
          email: 'admin@demo.gassigeher.local',
          password: process.env.DEMO_ADMIN_PASSWORD || 'demo1234'
        },
        // Demo users have fixed password
        greenUser: {
          email: 'anna@demo.gassigeher.local',
          password: 'demo1234'
        },
        orangeUser: {
          email: 'bernd@demo.gassigeher.local',
          password: 'demo1234'
        },
        blueUser: {
          email: 'clara@demo.gassigeher.local',
          password: 'demo1234'
        },
        // Central admin (platform-level)
        centralAdmin: {
          email: process.env.CENTRAL_ADMIN_EMAIL || 'admin@gassigeher.org',
          password: process.env.CENTRAL_ADMIN_PASSWORD || 'QKJPRpttNZ51cb92SEXxHCPwrwhDoBjB'
        }
      },

      // Page paths same as Simple-Mode (served from tenant subdomain)
      paths: {
        login: '/login.html',
        register: '/register.html',
        dashboard: '/dashboard.html',
        dogs: '/dogs.html',
        profile: '/profile.html',
        terms: '/terms.html',
        privacy: '/privacy.html',
        forgotPassword: '/forgot-password.html',
        adminDashboard: '/admin-dashboard.html',
        // SaaS-only paths (on main domain)
        landing: '/landing/',
        landingRegister: '/landing/register.html',
        central: '/central/'
      }
    }
  };

  return configs[mode] || configs[TEST_MODES.SAAS];
}

/**
 * Get test config from Playwright test info
 * Detects mode from project name (e.g., "simple-chromium" or "saas-chromium")
 */
function getConfigFromTestInfo(testInfo) {
  const projectName = testInfo?.project?.name || '';

  if (projectName.startsWith('simple')) {
    return getTestConfig(TEST_MODES.SIMPLE);
  }
  return getTestConfig(TEST_MODES.SAAS);
}

/**
 * Check if running in Simple-Mode
 */
function isSimpleMode(testInfo) {
  const config = getConfigFromTestInfo(testInfo);
  return config.mode === TEST_MODES.SIMPLE;
}

/**
 * Check if running in SaaS-Mode
 */
function isSaaSMode(testInfo) {
  const config = getConfigFromTestInfo(testInfo);
  return config.mode === TEST_MODES.SAAS;
}

module.exports = {
  TEST_MODES,
  getTestConfig,
  getConfigFromTestInfo,
  isSimpleMode,
  isSaaSMode
};
