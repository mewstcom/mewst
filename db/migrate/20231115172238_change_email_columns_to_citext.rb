# frozen_string_literal: true

class ChangeEmailColumnsToCitext < ActiveRecord::Migration[7.0]
  def change
    StrongMigrations.disable_check(:change_column)
    # standard:disable Rails/ReversibleMigration
    change_column :email_confirmations, :email, :citext
    change_column :users, :email, :citext
    # standard:enable Rails/ReversibleMigration
  end
end
