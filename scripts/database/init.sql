CREATE TABLE IF NOT EXISTS computers (
    id UUID PRIMARY KEY,
    mac_address VARCHAR(17) NOT NULL UNIQUE,
    computer_name VARCHAR(255) NOT NULL,
    ip_address VARCHAR(15) NOT NULL,
    employee_abbreviation VARCHAR(3),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on employee_abbreviation for faster lookups
CREATE INDEX IF NOT EXISTS idx_computers_employee_abbreviation ON computers (employee_abbreviation);

-- Create index on mac_address for faster lookups (redundant but explicit)
CREATE INDEX IF NOT EXISTS idx_computers_mac_address ON computers (mac_address);

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_computers_updated_at BEFORE UPDATE ON computers
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();