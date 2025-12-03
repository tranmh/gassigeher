const Database = require('better-sqlite3');
const TEST_PASSWORD_HASH = '$2a$10$LT4jdYaamd5Sxed9IhHTKuedmp/AvzGH27pJwCFzxAqAuO0c6OqfC';

function setupTestData(dbPath) {
    const db = new Database(dbPath);
    const now = new Date().toISOString();

    // Clear existing data (order matters due to foreign keys)
    db.exec('DELETE FROM blocked_dates');
    db.exec('DELETE FROM bookings');
    db.exec('DELETE FROM dogs');
    db.exec('DELETE FROM users');

    // Reset auto-increment so admin gets ID 1
    db.exec("DELETE FROM sqlite_sequence WHERE name IN ('users', 'dogs', 'blocked_dates', 'bookings')");

    const userStmt = db.prepare('INSERT INTO users (email, name, phone, password_hash, experience_level, is_verified, is_active, is_admin, is_super_admin, terms_accepted_at, last_activity_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)');
    const dogStmt = db.prepare('INSERT INTO dogs (name, breed, category, is_available, created_at) VALUES (?, ?, ?, ?, ?)');
    const blockedStmt = db.prepare('INSERT INTO blocked_dates (date, reason, created_by, created_at) VALUES (?, ?, ?, ?)');

    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate()+1);
    const blocked_date = tomorrow.toISOString().split('T')[0];

    // Admin must be first (gets ID 1 for blocked_dates FK)
    userStmt.run('admin@tierheim-goeppingen.de', 'Admin Name', null, TEST_PASSWORD_HASH, 'orange', 1, 1, 1, 1, now, now, now);
    userStmt.run('green@test.com', 'Green User', null, TEST_PASSWORD_HASH, 'green', 1, 1, 0, 0, now, now, now);
    userStmt.run('blue@test.com', 'Blue User', null, TEST_PASSWORD_HASH, 'blue', 1, 1, 0, 0, now, now, now);
    userStmt.run('delete-me@test.com', 'Delete Me User', null, TEST_PASSWORD_HASH, 'green', 1, 1, 0, 0, now, now, now);

    dogStmt.run('Rudolf', 'Chihuahua', 'green', 1, now);
    dogStmt.run('Max', 'Schäferhund', 'blue', 1, now);
    dogStmt.run('Ronny', 'Dalmatiner', 'orange', 1, now);

    blockedStmt.run(blocked_date, 'Feiertag', 1, now);

    db.close();
}

module.exports = { setupTestData };