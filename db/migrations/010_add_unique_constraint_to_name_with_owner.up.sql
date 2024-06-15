ALTER TABLE repositories
ADD CONSTRAINT unique_name_with_owner UNIQUE (name_with_owner);
