const sqlite3 = require('sqlite3').verbose();
const path = require('path');

/**
 * Database helper for E2E tests
 * Provides direct database access for:
 * - Setting up test data
 * - Bypassing email verification
 * - Cleaning up between tests
 */
class DBHelper {
  constructor(dbPath = '../test.db') {
    this.dbPath = path.resolve(__dirname, dbPath);
    this.db = null;
  }

  /**
   * Open database connection
   */
  async connect() {
    return new Promise((resolve, reject) => {
      this.db = new sqlite3.Database(this.dbPath, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  /**
   * Run SQL query
   */
  async run(sql, params = []) {
    return new Promise((resolve, reject) => {
      this.db.run(sql, params, function(err) {
        if (err) reject(err);
        else resolve({ lastID: this.lastID, changes: this.changes });
      });
    });
  }

  /**
   * Get single row
   */
  async get(sql, params = []) {
    return new Promise((resolve, reject) => {
      this.db.get(sql, params, (err, row) => {
        if (err) reject(err);
        else resolve(row);
      });
    });
  }

  /**
   * Get all rows
   */
  async all(sql, params = []) {
    return new Promise((resolve, reject) => {
      this.db.all(sql, params, (err, rows) => {
        if (err) reject(err);
        else resolve(rows);
      });
    });
  }

  /**
   * Create user
   * Password is bcrypt hash of "test123"
   */
  async createUser(userData) {
    const sql = `INSERT INTO users (
      email, name, password_hash, phone, experience_level,
      is_verified, is_active, terms_accepted_at, created_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`;

    const result = await this.run(sql, [
      userData.email,
      userData.name,
      // Bcrypt hash for "test123" (cost 10)
      '$2a$10$rZ5h8F5h5h5h5h5h5h5h5uX5h5h5h5h5h5h5h5h5h5h5h5h5h5h5h5',
      userData.phone || '+49 123 456789',
      userData.experience_level || 'green',
      userData.is_verified !== undefined ? userData.is_verified : 1,
      userData.is_active !== undefined ? userData.is_active : 1,
    ]);

    return result.lastID;
  }

  /**
   * Verify user (bypass email verification)
   */
  async verifyUser(email) {
    const sql = `UPDATE users SET is_verified = 1 WHERE email = ?`;
    await this.run(sql, [email]);
  }

  /**
   * Set verification token for user (for testing token flows)
   */
  async setVerificationToken(email, token, expiresAt = null) {
    const expires = expiresAt || new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    const sql = `UPDATE users SET verification_token = ?, verification_token_expires_at = ? WHERE email = ?`;
    await this.run(sql, [token, expires, email]);
  }

  /**
   * Set password reset token
   */
  async setPasswordResetToken(email, token, expiresAt = null) {
    const expires = expiresAt || new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString();
    const sql = `UPDATE users SET password_reset_token = ?, password_reset_token_expires_at = ? WHERE email = ?`;
    await this.run(sql, [token, expires, email]);
  }

  /**
   * Get user by email
   */
  async getUserByEmail(email) {
    const sql = `SELECT * FROM users WHERE email = ?`;
    return await this.get(sql, [email]);
  }

  /**
   * Create dog
   */
  async createDog(dogData) {
    const sql = `INSERT INTO dogs (
      name, breed, size, age, category,
      is_available, unavailable_reason, created_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`;

    const result = await this.run(sql, [
      dogData.name,
      dogData.breed || 'Mixed Breed',
      dogData.size || 'medium',
      dogData.age || 3,
      dogData.category || 'green',
      dogData.is_available !== undefined ? dogData.is_available : 1,
      dogData.unavailable_reason || null,
    ]);

    return result.lastID;
  }

  /**
   * Create booking
   */
  async createBooking(bookingData) {
    const sql = `INSERT INTO bookings (
      user_id, dog_id, date, walk_type, scheduled_time,
      status, created_at
    ) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`;

    const result = await this.run(sql, [
      bookingData.user_id,
      bookingData.dog_id,
      bookingData.date,
      bookingData.walk_type || 'morning',
      bookingData.scheduled_time || '09:00',
      bookingData.status || 'scheduled',
    ]);

    return result.lastID;
  }

  /**
   * Block date
   */
  async blockDate(date, reason = 'Test block') {
    const sql = `INSERT INTO blocked_dates (date, reason, created_at)
                 VALUES (?, ?, datetime('now'))`;
    const result = await this.run(sql, [date, reason]);
    return result.lastID;
  }

  /**
   * Update system setting
   */
  async updateSetting(key, value) {
    const sql = `UPDATE system_settings SET value = ? WHERE key = ?`;
    await this.run(sql, [value, key]);
  }

  /**
   * Reset database (delete all data but keep schema)
   */
  async resetDatabase() {
    const tables = [
      'bookings',
      'reactivation_requests',
      'blocked_dates',
      'dogs',
      'users',
    ];

    for (const table of tables) {
      await this.run(`DELETE FROM ${table}`);
    }

    // Reset settings to defaults
    await this.run(`UPDATE system_settings SET value = '14' WHERE key = 'booking_advance_days'`);
    await this.run(`UPDATE system_settings SET value = '12' WHERE key = 'cancellation_notice_hours'`);
    await this.run(`UPDATE system_settings SET value = '365' WHERE key = 'auto_deactivation_days'`);
  }

  /**
   * Close database connection
   */
  close() {
    if (this.db) {
      this.db.close();
    }
  }
}

module.exports = DBHelper;

// DONE: Database helper utility for direct database access in tests
