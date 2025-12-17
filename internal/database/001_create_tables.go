package database

func init() {
	RegisterMigration(&Migration{
		ID:          "001_create_tables",
		Description: "Create all database tables with final schema",
		Up: map[string]string{
			"sqlite": `
-- Users table (without experience_level, without name - use first_name/last_name)
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  first_name TEXT,
  last_name TEXT,
  email TEXT UNIQUE,
  phone TEXT,
  password_hash TEXT,
  is_verified INTEGER DEFAULT 0,
  is_active INTEGER DEFAULT 1,
  is_deleted INTEGER DEFAULT 0,
  is_admin INTEGER DEFAULT 0,
  is_super_admin INTEGER DEFAULT 0,
  must_change_password INTEGER DEFAULT 0,
  verification_token TEXT,
  verification_token_expires TIMESTAMP,
  password_reset_token TEXT,
  password_reset_expires TIMESTAMP,
  profile_photo TEXT,
  anonymous_id TEXT UNIQUE,
  terms_accepted_at TIMESTAMP NOT NULL,
  last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  deactivated_at TIMESTAMP,
  deactivation_reason TEXT,
  reactivated_at TIMESTAMP,
  deleted_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_last_activity ON users(last_activity_at, is_active);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_admin ON users(is_admin);
CREATE INDEX IF NOT EXISTS idx_users_super_admin ON users(is_super_admin);
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_super_admin ON users(is_super_admin) WHERE is_super_admin = 1;

-- Color categories table (must be created before dogs and user_colors due to FK)
CREATE TABLE IF NOT EXISTS color_categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  hex_code TEXT NOT NULL,
  pattern_icon TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_color_categories_sort ON color_categories(sort_order);

-- Dogs table (without category - use color_id)
CREATE TABLE IF NOT EXISTS dogs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  breed TEXT NOT NULL,
  size TEXT CHECK(size IN ('small', 'medium', 'large')),
  age INTEGER,
  color_id INTEGER REFERENCES color_categories(id),
  photo TEXT,
  photo_thumbnail TEXT,
  special_needs TEXT,
  pickup_location TEXT,
  walk_route TEXT,
  walk_duration INTEGER,
  special_instructions TEXT,
  default_morning_time TEXT,
  default_evening_time TEXT,
  is_available INTEGER DEFAULT 1,
  is_featured INTEGER DEFAULT 0,
  unavailable_reason TEXT,
  unavailable_since TIMESTAMP,
  external_link TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_dogs_available ON dogs(is_available);
CREATE INDEX IF NOT EXISTS idx_dogs_color ON dogs(color_id);
CREATE INDEX IF NOT EXISTS idx_dogs_featured ON dogs(is_featured);

-- Bookings table (without walk_type - use only scheduled_time)
CREATE TABLE IF NOT EXISTS bookings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  dog_id INTEGER NOT NULL,
  date DATE NOT NULL,
  scheduled_time TEXT NOT NULL,
  status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at TIMESTAMP,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  requires_approval INTEGER DEFAULT 0,
  approval_status TEXT DEFAULT 'approved',
  approved_by INTEGER,
  approved_at TIMESTAMP,
  rejection_reason TEXT,
  reminder_sent_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE(dog_id, date, scheduled_time)
);
CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);

-- Blocked dates table (with dog_id for dog-specific blocking)
CREATE TABLE IF NOT EXISTS blocked_dates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date DATE NOT NULL,
  dog_id INTEGER,
  reason TEXT NOT NULL,
  created_by INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id),
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_dates_dog_date ON blocked_dates (COALESCE(dog_id, 0), date);
CREATE INDEX IF NOT EXISTS idx_blocked_dates_date ON blocked_dates (date);

-- Experience requests table (for level promotion workflow)
CREATE TABLE IF NOT EXISTS experience_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  requested_level TEXT CHECK(requested_level IN ('blue', 'orange')),
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- System settings table (key-value store)
CREATE TABLE IF NOT EXISTS system_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reactivation requests table
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_reactivation_pending ON reactivation_requests(status, created_at);

-- Booking time rules table
CREATE TABLE IF NOT EXISTS booking_time_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  day_type TEXT NOT NULL,
  rule_name TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  is_blocked INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(day_type, rule_name)
);

-- Custom holidays table
CREATE TABLE IF NOT EXISTS custom_holidays (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  source TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by INTEGER,
  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);

-- Feiertage cache table (German holiday API cache)
CREATE TABLE IF NOT EXISTS feiertage_cache (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  year INTEGER NOT NULL UNIQUE,
  state TEXT NOT NULL,
  data TEXT NOT NULL,
  fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL
);

-- Walk reports table
CREATE TABLE IF NOT EXISTS walk_reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  booking_id INTEGER NOT NULL UNIQUE,
  behavior_rating INTEGER NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level TEXT NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_walk_reports_booking_id ON walk_reports(booking_id);

-- Walk report photos table
CREATE TABLE IF NOT EXISTS walk_report_photos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  walk_report_id INTEGER NOT NULL,
  photo_path TEXT NOT NULL,
  photo_thumbnail TEXT NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_report_id ON walk_report_photos(walk_report_id);

-- User colors junction table (many-to-many user-color relationship)
CREATE TABLE IF NOT EXISTS user_colors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  granted_by INTEGER,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id),
  UNIQUE(user_id, color_id)
);
CREATE INDEX IF NOT EXISTS idx_user_colors_user ON user_colors(user_id);
CREATE INDEX IF NOT EXISTS idx_user_colors_color ON user_colors(color_id);

-- Color requests table (for user color category requests)
CREATE TABLE IF NOT EXISTS color_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_color_requests_user ON color_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_color_requests_status ON color_requests(status);
CREATE INDEX IF NOT EXISTS idx_color_requests_color ON color_requests(color_id);
`,
			"mysql": `
-- Users table (without experience_level, without name - use first_name/last_name)
CREATE TABLE IF NOT EXISTS users (
  id INT AUTO_INCREMENT PRIMARY KEY,
  first_name VARCHAR(255),
  last_name VARCHAR(255),
  email VARCHAR(255) UNIQUE,
  phone VARCHAR(50),
  password_hash VARCHAR(255),
  is_verified TINYINT(1) DEFAULT 0,
  is_active TINYINT(1) DEFAULT 1,
  is_deleted TINYINT(1) DEFAULT 0,
  is_admin TINYINT(1) DEFAULT 0,
  is_super_admin TINYINT(1) DEFAULT 0,
  must_change_password TINYINT(1) DEFAULT 0,
  verification_token VARCHAR(255),
  verification_token_expires DATETIME,
  password_reset_token VARCHAR(255),
  password_reset_expires DATETIME,
  profile_photo VARCHAR(255),
  anonymous_id VARCHAR(255) UNIQUE,
  terms_accepted_at DATETIME NOT NULL,
  last_activity_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  deactivated_at DATETIME,
  deactivation_reason TEXT,
  reactivated_at DATETIME,
  deleted_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_users_last_activity (last_activity_at, is_active),
  INDEX idx_users_email (email),
  INDEX idx_users_admin (is_admin),
  INDEX idx_users_super_admin (is_super_admin)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Color categories table
CREATE TABLE IF NOT EXISTS color_categories (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL UNIQUE,
  hex_code VARCHAR(20) NOT NULL,
  pattern_icon VARCHAR(50),
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_color_categories_sort (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dogs table (without category - use color_id)
CREATE TABLE IF NOT EXISTS dogs (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  breed VARCHAR(255) NOT NULL,
  size VARCHAR(20) CHECK(size IN ('small', 'medium', 'large')),
  age INT,
  color_id INT,
  photo VARCHAR(255),
  photo_thumbnail VARCHAR(255),
  special_needs TEXT,
  pickup_location VARCHAR(255),
  walk_route TEXT,
  walk_duration INT,
  special_instructions TEXT,
  default_morning_time VARCHAR(10),
  default_evening_time VARCHAR(10),
  is_available TINYINT(1) DEFAULT 1,
  is_featured TINYINT(1) DEFAULT 0,
  unavailable_reason TEXT,
  unavailable_since DATETIME,
  external_link VARCHAR(500),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_dogs_available (is_available),
  INDEX idx_dogs_color (color_id),
  INDEX idx_dogs_featured (is_featured),
  FOREIGN KEY (color_id) REFERENCES color_categories(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bookings table (without walk_type)
CREATE TABLE IF NOT EXISTS bookings (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  dog_id INT NOT NULL,
  date DATE NOT NULL,
  scheduled_time VARCHAR(10) NOT NULL,
  status VARCHAR(20) DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at DATETIME,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  requires_approval TINYINT(1) DEFAULT 0,
  approval_status VARCHAR(20) DEFAULT 'approved',
  approved_by INT,
  approved_at DATETIME,
  rejection_reason TEXT,
  reminder_sent_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE KEY unique_dog_date_time (dog_id, date, scheduled_time),
  INDEX idx_bookings_user (user_id),
  INDEX idx_bookings_dog (dog_id),
  INDEX idx_bookings_date (date),
  INDEX idx_bookings_status (status),
  INDEX idx_bookings_approval_status (approval_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Blocked dates table
CREATE TABLE IF NOT EXISTS blocked_dates (
  id INT AUTO_INCREMENT PRIMARY KEY,
  date DATE NOT NULL,
  dog_id INT,
  reason TEXT NOT NULL,
  created_by INT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  dog_id_unique INT AS (COALESCE(dog_id, 0)) STORED,
  FOREIGN KEY (created_by) REFERENCES users(id),
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  UNIQUE KEY idx_blocked_dates_dog_date (dog_id_unique, date),
  INDEX idx_blocked_dates_date (date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Experience requests table
CREATE TABLE IF NOT EXISTS experience_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  requested_level VARCHAR(20) CHECK(requested_level IN ('blue', 'orange')),
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- System settings table
CREATE TABLE IF NOT EXISTS system_settings (
  ` + "`key`" + ` VARCHAR(255) PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Reactivation requests table
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id),
  INDEX idx_reactivation_pending (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Booking time rules table
CREATE TABLE IF NOT EXISTS booking_time_rules (
  id INT AUTO_INCREMENT PRIMARY KEY,
  day_type VARCHAR(20) NOT NULL,
  rule_name VARCHAR(100) NOT NULL,
  start_time VARCHAR(10) NOT NULL,
  end_time VARCHAR(10) NOT NULL,
  is_blocked TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY unique_day_rule (day_type, rule_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Custom holidays table
CREATE TABLE IF NOT EXISTS custom_holidays (
  id INT AUTO_INCREMENT PRIMARY KEY,
  date DATE NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  source VARCHAR(20) NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_by INT,
  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_custom_holidays_date (date),
  INDEX idx_custom_holidays_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Feiertage cache table
CREATE TABLE IF NOT EXISTS feiertage_cache (
  id INT AUTO_INCREMENT PRIMARY KEY,
  year INT NOT NULL UNIQUE,
  state VARCHAR(10) NOT NULL,
  data TEXT NOT NULL,
  fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Walk reports table
CREATE TABLE IF NOT EXISTS walk_reports (
  id INT AUTO_INCREMENT PRIMARY KEY,
  booking_id INT NOT NULL UNIQUE,
  behavior_rating INT NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level VARCHAR(10) NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE,
  INDEX idx_walk_reports_booking_id (booking_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Walk report photos table
CREATE TABLE IF NOT EXISTS walk_report_photos (
  id INT AUTO_INCREMENT PRIMARY KEY,
  walk_report_id INT NOT NULL,
  photo_path VARCHAR(255) NOT NULL,
  photo_thumbnail VARCHAR(255) NOT NULL,
  display_order INT NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE,
  INDEX idx_walk_report_photos_report_id (walk_report_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- User colors junction table
CREATE TABLE IF NOT EXISTS user_colors (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  color_id INT NOT NULL,
  granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  granted_by INT,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id),
  UNIQUE KEY unique_user_color (user_id, color_id),
  INDEX idx_user_colors_user (user_id),
  INDEX idx_user_colors_color (color_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Color requests table
CREATE TABLE IF NOT EXISTS color_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  color_id INT NOT NULL,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id),
  INDEX idx_color_requests_user (user_id),
  INDEX idx_color_requests_status (status),
  INDEX idx_color_requests_color (color_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
-- Users table (without experience_level, without name - use first_name/last_name)
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  first_name VARCHAR(255),
  last_name VARCHAR(255),
  email VARCHAR(255) UNIQUE,
  phone VARCHAR(50),
  password_hash VARCHAR(255),
  is_verified BOOLEAN DEFAULT FALSE,
  is_active BOOLEAN DEFAULT TRUE,
  is_deleted BOOLEAN DEFAULT FALSE,
  is_admin BOOLEAN DEFAULT FALSE,
  is_super_admin BOOLEAN DEFAULT FALSE,
  must_change_password BOOLEAN DEFAULT FALSE,
  verification_token VARCHAR(255),
  verification_token_expires TIMESTAMP WITH TIME ZONE,
  password_reset_token VARCHAR(255),
  password_reset_expires TIMESTAMP WITH TIME ZONE,
  profile_photo VARCHAR(255),
  anonymous_id VARCHAR(255) UNIQUE,
  terms_accepted_at TIMESTAMP WITH TIME ZONE NOT NULL,
  last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  deactivated_at TIMESTAMP WITH TIME ZONE,
  deactivation_reason TEXT,
  reactivated_at TIMESTAMP WITH TIME ZONE,
  deleted_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_last_activity ON users(last_activity_at, is_active);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_admin ON users(is_admin);
CREATE INDEX IF NOT EXISTS idx_users_super_admin ON users(is_super_admin);
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_super_admin ON users(is_super_admin) WHERE is_super_admin = TRUE;

-- Color categories table
CREATE TABLE IF NOT EXISTS color_categories (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL UNIQUE,
  hex_code VARCHAR(20) NOT NULL,
  pattern_icon VARCHAR(50),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_color_categories_sort ON color_categories(sort_order);

-- Dogs table (without category - use color_id)
CREATE TABLE IF NOT EXISTS dogs (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  breed VARCHAR(255) NOT NULL,
  size VARCHAR(20) CHECK(size IN ('small', 'medium', 'large')),
  age INTEGER,
  color_id INTEGER REFERENCES color_categories(id),
  photo VARCHAR(255),
  photo_thumbnail VARCHAR(255),
  special_needs TEXT,
  pickup_location VARCHAR(255),
  walk_route TEXT,
  walk_duration INTEGER,
  special_instructions TEXT,
  default_morning_time VARCHAR(10),
  default_evening_time VARCHAR(10),
  is_available BOOLEAN DEFAULT TRUE,
  is_featured BOOLEAN DEFAULT FALSE,
  unavailable_reason TEXT,
  unavailable_since TIMESTAMP WITH TIME ZONE,
  external_link VARCHAR(500),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_dogs_available ON dogs(is_available);
CREATE INDEX IF NOT EXISTS idx_dogs_color ON dogs(color_id);
CREATE INDEX IF NOT EXISTS idx_dogs_featured ON dogs(is_featured);

-- Bookings table (without walk_type)
CREATE TABLE IF NOT EXISTS bookings (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  dog_id INTEGER NOT NULL,
  date DATE NOT NULL,
  scheduled_time VARCHAR(10) NOT NULL,
  status VARCHAR(20) DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at TIMESTAMP WITH TIME ZONE,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  requires_approval BOOLEAN DEFAULT FALSE,
  approval_status VARCHAR(20) DEFAULT 'approved',
  approved_by INTEGER,
  approved_at TIMESTAMP WITH TIME ZONE,
  rejection_reason TEXT,
  reminder_sent_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE(dog_id, date, scheduled_time)
);
CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);

-- Blocked dates table
CREATE TABLE IF NOT EXISTS blocked_dates (
  id SERIAL PRIMARY KEY,
  date DATE NOT NULL,
  dog_id INTEGER,
  reason TEXT NOT NULL,
  created_by INTEGER NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id),
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_dates_dog_date ON blocked_dates (COALESCE(dog_id, 0), date);
CREATE INDEX IF NOT EXISTS idx_blocked_dates_date ON blocked_dates (date);

-- Experience requests table
CREATE TABLE IF NOT EXISTS experience_requests (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  requested_level VARCHAR(20) CHECK(requested_level IN ('blue', 'orange')),
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- System settings table
CREATE TABLE IF NOT EXISTS system_settings (
  key VARCHAR(255) PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Reactivation requests table
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_reactivation_pending ON reactivation_requests(status, created_at);

-- Booking time rules table
CREATE TABLE IF NOT EXISTS booking_time_rules (
  id SERIAL PRIMARY KEY,
  day_type VARCHAR(20) NOT NULL,
  rule_name VARCHAR(100) NOT NULL,
  start_time VARCHAR(10) NOT NULL,
  end_time VARCHAR(10) NOT NULL,
  is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(day_type, rule_name)
);

-- Custom holidays table
CREATE TABLE IF NOT EXISTS custom_holidays (
  id SERIAL PRIMARY KEY,
  date DATE NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  source VARCHAR(20) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  created_by INTEGER,
  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);

-- Feiertage cache table
CREATE TABLE IF NOT EXISTS feiertage_cache (
  id SERIAL PRIMARY KEY,
  year INTEGER NOT NULL UNIQUE,
  state VARCHAR(10) NOT NULL,
  data TEXT NOT NULL,
  fetched_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Walk reports table
CREATE TABLE IF NOT EXISTS walk_reports (
  id SERIAL PRIMARY KEY,
  booking_id INTEGER NOT NULL UNIQUE,
  behavior_rating INTEGER NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level VARCHAR(10) NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_walk_reports_booking_id ON walk_reports(booking_id);

-- Walk report photos table
CREATE TABLE IF NOT EXISTS walk_report_photos (
  id SERIAL PRIMARY KEY,
  walk_report_id INTEGER NOT NULL,
  photo_path VARCHAR(255) NOT NULL,
  photo_thumbnail VARCHAR(255) NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_report_id ON walk_report_photos(walk_report_id);

-- User colors junction table
CREATE TABLE IF NOT EXISTS user_colors (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  granted_by INTEGER,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id),
  UNIQUE(user_id, color_id)
);
CREATE INDEX IF NOT EXISTS idx_user_colors_user ON user_colors(user_id);
CREATE INDEX IF NOT EXISTS idx_user_colors_color ON user_colors(color_id);

-- Color requests table
CREATE TABLE IF NOT EXISTS color_requests (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_color_requests_user ON color_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_color_requests_status ON color_requests(status);
CREATE INDEX IF NOT EXISTS idx_color_requests_color ON color_requests(color_id);
`,
		},
	})
}
