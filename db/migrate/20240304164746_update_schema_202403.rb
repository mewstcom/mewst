# frozen_string_literal: true

class UpdateSchema202403 < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:rename_column)
    rename_column :profiles, :profileable_type, :owner_type

    create_table :user_profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :user, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.timestamps
    end

    remove_index :actors, :profile_id
    remove_index :actors, :user_id

    StrongMigrations.disable_check(:remove_column)
    remove_column :users, :profile_id # standard:disable Rails/ReversibleMigration
  end
end
