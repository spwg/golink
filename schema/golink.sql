create table if not exists links (
    id integer primary key,
    name text not null unique,
    url text not null
);
