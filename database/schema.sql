-- Users
CREATE TABLE IF NOT EXISTS users (
    username    VARCHAR(50)  PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    email       VARCHAR(100) NOT NULL UNIQUE,
    password    MEDIUMTEXT NOT NULL,
    user_type   ENUM('user','admin')  NOT NULL DEFAULT 'user',
    created_on  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Profiles (picture path per user)
CREATE TABLE IF NOT EXISTS profiles (
    username    VARCHAR(50),
    path        VARCHAR(255) NOT NULL,
    PRIMARY KEY(username),
    CONSTRAINT profiles_users
    FOREIGN KEY(username)
    REFERENCES users(username) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE
);

-- Problems
CREATE TABLE IF NOT EXISTS problems (
    id          	BIGINT UNSIGNED AUTO_INCREMENT       PRIMARY KEY,
    title       	VARCHAR(200) NOT NULL,
    statement   	TEXT,
    input       	TEXT,
    output      	TEXT,
    constraints 	TEXT,
    author 			VARCHAR(50)	NOT NULL,
    time_limit 	 	INT	 DEFAULT 1,
    memory_limit 	INT	 DEFAULT 512,
    editorial 		TEXT,
    editorial_code 			TEXT,
    visibility 		BOOLEAN		NOT NULL DEFAULT FALSE,
    CONSTRAINT problems_users
    FOREIGN KEY(author) 
    REFERENCES users(username)
    ON DELETE CASCADE 
    ON UPDATE CASCADE
);

-- Test Cases
CREATE TABLE IF NOT EXISTS test_cases (
    id          BIGINT UNSIGNED AUTO_INCREMENT  PRIMARY KEY,
    problem_id  BIGINT UNSIGNED NOT NULL,
    input       TEXT    NOT NULL,
    output      TEXT    NOT NULL,
    type		ENUM('sample','hidden') NOT NULL DEFAULT 'hidden',
    CONSTRAINT test_cases_problems
    FOREIGN KEY(problem_id)
    REFERENCES problems(id)
    ON DELETE CASCADE 
    ON UPDATE CASCADE
);

-- Tags
CREATE TABLE IF NOT EXISTS tags(
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);
INSERT IGNORE INTO tags (name) VALUES 
('2-sat'),
('binary search'),
('bitmasks'),
('brute force'),
('chinese remainder theorem'),
('combinatorics'),
('constructive algorithms'),
('data structures'),
('dfs and similar'),
('divide and conquer'),
('dp'),
('dsu'),
('expression parsing'),
('fft'),
('flows'),
('games'),
('geometry'),
('graph matchings'),
('graphs'),
('greedy'),
('hashing'),
('implementation'),
('interactive'),
('math'),
('matrices'),
('meet-in-the-middle'),
('number theory'),
('probabilities'),
('schedules'),
('shortest paths'),
('sortings'),
('string suffix structures'),
('strings'),
('ternary search'),
('trees'),
('two pointers');

-- Tags of a Problem
CREATE TABLE IF NOT EXISTS problem_tags (
    problem_id  BIGINT UNSIGNED NOT NULL,        
    tag_id INT  NOT NULL,
    PRIMARY KEY(problem_id,tag_id),

    CONSTRAINT problem_tags_problems
    FOREIGN KEY(problem_id)
    REFERENCES problems(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE,

    CONSTRAINT problem_tags_tags
    FOREIGN KEY(tag_id)
    REFERENCES tags(id)
    ON DELETE CASCADE 
    ON UPDATE CASCADE
);

-- Ratings
CREATE TABLE IF NOT EXISTS ratings (
    problem_id  BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    rating      INT NOT NULL DEFAULT 800,
    CONSTRAINT ratings_problems
    FOREIGN KEY(problem_id)
    REFERENCES problems(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);


-- Contests
CREATE TABLE IF NOT EXISTS contests (
    id          BIGINT UNSIGNED AUTO_INCREMENT       PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    start_time  DATETIME NOT NULL,
	end_time 	DATETIME NOT NULL,
    visibility	BOOLEAN  NOT NULL DEFAULT FALSE
);

-- Tasks (problems assigned to a contest with a score)
CREATE TABLE IF NOT EXISTS tasks (
    problem_id  BIGINT UNSIGNED,
    contest_id  BIGINT UNSIGNED,
    score       INT NOT NULL DEFAULT 1000,
    PRIMARY KEY(problem_id,contest_id),
    CONSTRAINT tasks_problems
    FOREIGN KEY(problem_id)
    REFERENCES problems(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
    
    CONSTRAINT tasks_contests
    FOREIGN KEY(contest_id)
    REFERENCES contests(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

-- Participants (users enrolled in a contest)
CREATE TABLE IF NOT EXISTS participants (
    contest_id  BIGINT UNSIGNED,
    participant    VARCHAR(50),
    PRIMARY KEY (contest_id, participant),
    
    CONSTRAINT participants_contests
    FOREIGN KEY(contest_id)
    REFERENCES contests(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
    
    CONSTRAINT participants_users
    FOREIGN KEY(participant)
    REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

-- Authors (users who authored / set a contest)
CREATE TABLE IF NOT EXISTS authors (
    contest_id  BIGINT UNSIGNED,
    author    VARCHAR(50),
    role 		ENUM('owner','moderator') NOT NULL DEFAULT 'moderator',
    PRIMARY KEY (contest_id, author),
    
    CONSTRAINT authors_contests
    FOREIGN KEY(contest_id)
    REFERENCES contests(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
    
    CONSTRAINT authors_users
    FOREIGN KEY(author)
    REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

-- Submissions
CREATE TABLE IF NOT EXISTS submissions (
    id           BIGINT UNSIGNED AUTO_INCREMENT      PRIMARY KEY NOT NULL,
    problem_id   BIGINT UNSIGNED NOT NULL,
    username     VARCHAR(50) NOT NULL,
    code         TEXT        NOT NULL,
    language     VARCHAR(50) NOT NULL DEFAULT 'GNU G++14',
    submitted_at DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verdict      VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    CONSTRAINT submissions_problems
    FOREIGN KEY(problem_id)
    REFERENCES problems(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
    
    CONSTRAINT submissions_users
    FOREIGN KEY(username)
    REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

-- Streaks (daily problem-solving streaks per user)
CREATE TABLE IF NOT EXISTS streaks (
    username    VARCHAR(50) NOT NULL,
    problem_id  BIGINT UNSIGNED NOT NULL,
    date        DATE NOT NULL DEFAULT (UTC_DATE()),
    PRIMARY KEY (username, problem_id),
    CONSTRAINT streaks_users
    FOREIGN KEY(username)
    REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
    
    CONSTRAINT streaks_problems
    FOREIGN KEY(problem_id)
    REFERENCES problems(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);


ALTER TABLE submissions
ADD INDEX IF NOT EXISTS idx_submissions_user (username),
ADD INDEX IF NOT EXISTS idx_submissions_problem (problem_id);

ALTER TABLE tasks
ADD INDEX IF NOT EXISTS idx_tasks_contest (contest_id);

ALTER TABLE participants
ADD INDEX IF NOT EXISTS idx_participants_user (participant);
