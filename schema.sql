CREATE DATABASE IF NOT EXISTS task_manager;

USE task_manager;

CREATE TABLE IF NOT EXISTS tasks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM('pending', 'completed') NOT NULL DEFAULT 'pending',
    priority ENUM('low', 'medium', 'high') NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

