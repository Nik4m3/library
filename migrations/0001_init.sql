-- +goose Up
-- +goose StatementBegin
create table if not exists reading_rooms
(
    id   uuid default gen_random_uuid() primary key,
    name varchar not null unique
);
insert into reading_rooms (name)
values ('Зал художественной литературы'),
       ('Зал технической литературы'),
       ('Зал иностранной литературы');

create table if not exists book_groups
(
    id   uuid default gen_random_uuid() primary key,
    name varchar not null unique
);
insert into book_groups (name)
values ('Живописные рассказы'),
       ('Программирование'),
       ('Исторические рассказы');

create table if not exists users
(
    id            uuid default gen_random_uuid() primary key,
    name          varchar not null,
    date_birth    date    not null,
    ticket_number serial  not null,
    phone         varchar(16),
    CONSTRAINT chk_phone_mask CHECK (phone IS NULL OR phone ~ '^\+7 \(\d{3}\) \d{7}$')
);
insert into users (name, date_birth, phone)
values ('Сергеев Сергей Сергеевич', '05-22-1990', '+7 (951) 1234567'),
       ('Антонов Антон Валерьевич', '11-01-1992', '+7 (951) 7654321');

create table if not exists book_authors
(
    id   uuid default gen_random_uuid() primary key,
    name varchar not null unique
);
insert into book_authors (name)
values ('Пушкин А.С.'),
       ('Буч'),
       ('Andrew Hunt');

create table if not exists place_publications
(
    id   uuid default gen_random_uuid() primary key,
    name varchar not null unique
);
insert into place_publications(name)
values ('Моссква'),
       ('Челябинск'),
       ('New York');

create table if not exists publishing_houses
(
    id   uuid default gen_random_uuid() primary key,
    name varchar not null unique
);
insert into publishing_houses(name)
values ('Альфа'),
       ('2-комсомольца'),
       ('Бетта');

create table if not exists books
(
    id                   uuid    default gen_random_uuid() primary key,
    name                 varchar                                 not null,
    reading_room_id      uuid references reading_rooms (id)      not null,
    author_id            uuid references book_authors (id)       not null,
    place_publication_id uuid references place_publications (id) not null,
    published_house_id   uuid references publishing_houses (id)  not null,
    year_publication     integer                                 not null,
    book_group_id        uuid references book_groups (id)        not null,
    pages                integer                                 not null CHECK (pages > 0),
    number_copies        integer default 1                       not null CHECK (number_copies >= 1),
    constraint unique_book UNIQUE (name, author_id, year_publication)
    );
CREATE INDEX IF NOT EXISTS idx_books_author ON books (author_id);
CREATE INDEX IF NOT EXISTS idx_books_group ON books (book_group_id);
CREATE INDEX IF NOT EXISTS idx_books_room ON books (reading_room_id);
CREATE INDEX IF NOT EXISTS idx_books_place ON books (place_publication_id);
CREATE INDEX IF NOT EXISTS idx_books_publisher ON books (published_house_id);

create table if not exists accounting_books
(
    id          uuid default gen_random_uuid() primary key,
    user_id     uuid references users (id),
    book_id     uuid               not null,
    date_issue  date default now() not null,
    date_return date,
    CONSTRAINT chk_return_not_before_issue CHECK (date_return IS NULL OR date_return >= date_issue)
    );

CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_active_book
    ON accounting_books (user_id, book_id) WHERE date_return IS NULL;

CREATE INDEX IF NOT EXISTS idx_loans_issue_date
    ON accounting_books (date_issue);
CREATE INDEX IF NOT EXISTS idx_books_room_year
    ON books (reading_room_id, year_publication);
CREATE INDEX IF NOT EXISTS idx_books_room_group
    ON books (reading_room_id, book_group_id);

CREATE OR REPLACE FUNCTION book_conditions_constraints()
    RETURNS trigger
    LANGUAGE plpgsql
AS
$$
DECLARE
user_cnt   INT;
    book_cnt   INT;
    max_copies INT;
BEGIN
SELECT number_copies
INTO max_copies
FROM books
WHERE id = NEW.book_id
    FOR UPDATE;

SELECT COUNT(*)
INTO user_cnt
FROM accounting_books
WHERE user_id = NEW.user_id
  AND date_return IS NULL;

SELECT COUNT(*)
INTO book_cnt
FROM accounting_books
WHERE book_id = NEW.book_id
  AND date_return IS NULL;

IF user_cnt >= 5 THEN
        RAISE EXCEPTION 'Пользователь уже держит 5 или более книг';
END IF;
    IF book_cnt >= max_copies THEN
        RAISE EXCEPTION 'Свободных экземпляров книги больше нет';
END IF;
    IF EXISTS (SELECT 1
               FROM accounting_books ab
                        JOIN books b ON b.id = ab.book_id
               WHERE ab.user_id = NEW.user_id
                 AND ab.date_return IS NULL
                 AND b.id = NEW.book_id) THEN
        RAISE EXCEPTION 'Нельзя выдать второй экземпляр той же книги одному пользователю одновременно';
END IF;
RETURN NEW;
END;
$$;

CREATE TRIGGER trg_book_conditions_constraints
    BEFORE INSERT OR UPDATE
                         ON accounting_books
                         FOR EACH ROW
                         WHEN (NEW.date_return IS NULL)
                         EXECUTE FUNCTION book_conditions_constraints();


INSERT INTO books (name, reading_room_id, author_id, place_publication_id, published_house_id,
                   year_publication, book_group_id, pages, number_copies)
SELECT 'Евгений Онегин',
       (SELECT id FROM reading_rooms WHERE name = 'Зал художественной литературы'),
       (SELECT id FROM book_authors WHERE name = 'Пушкин А.С.'),
       (SELECT id FROM place_publications WHERE name = 'Моссква'),
       (SELECT id FROM publishing_houses WHERE name = 'Альфа'),
       1833,
       (SELECT id FROM book_groups WHERE name = 'Живописные рассказы'),
       320,
       2
    WHERE NOT EXISTS (SELECT 1
                  FROM books b
                  WHERE b.name = 'Евгений Онегин'
                    AND b.author_id = (SELECT id FROM book_authors WHERE name = 'Пушкин А.С.')
                    AND b.year_publication = 1833);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop trigger if exists trg_book_conditions_constraints on accounting_books;
drop function if exists book_conditions_constraints();
drop index if exists uniq_user_active_book;
drop table if exists accounting_books;
drop table if exists books;
drop table if exists publishing_houses;
drop table if exists place_publications;
drop table if exists book_authors;
drop table if exists users;
drop table if exists book_groups;
drop table if exists reading_rooms;
-- +goose StatementEnd
