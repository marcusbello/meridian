

--name: Create listing table
--description: This migration creates the listings table with the necessary fields.
CREATE TABLE listings (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(15,2) NOT NULL,
    category VARCHAR(50),
    pictures_url TEXT,
    negotiable BOOLEAN DEFAULT FALSE,
    type VARCHAR(50),
    location VARCHAR(100),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    featured BOOLEAN DEFAULT FALSE,
    user_id INT NOT NULL
);