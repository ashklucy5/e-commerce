-- Modify "support_actors" table
ALTER TABLE "support_actors" DROP CONSTRAINT "support_actors_staff_account_id_fkey", ADD CONSTRAINT "support_actors_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
