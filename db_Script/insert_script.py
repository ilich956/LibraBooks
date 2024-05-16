import psycopg2

# Connect to your PostgreSQL database
conn = psycopg2.connect(
    dbname="adv_database",
    user="postgres",
    password="bayipket",
    host="localhost",
    port="5432"
)

# Open a cursor to perform database operations
cur = conn.cursor()

# Read emails from the .txt file
with open('emails.txt', 'r') as file:
    emails = file.readlines()

# Insert each email into the user_table
for email in emails:
    try:
        cur.execute("INSERT INTO user_table (email) VALUES (%s)", (email.strip(),))
    except psycopg2.Error as e:
        print("Error inserting email:", e)
        conn.rollback()  # Rollback the transaction in case of error
    else:
        print("Email inserted successfully:", email.strip())

# Commit the transaction
conn.commit()

# Close the cursor and connection
cur.close()
conn.close()
