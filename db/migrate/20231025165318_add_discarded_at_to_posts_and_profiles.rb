# frozen_string_literal: true

class AddDiscardedAtToPostsAndProfiles < ActiveRecord::Migration[7.0]
  disable_ddl_transaction!

  def change
    %i[posts profiles].each do |table_name|
      add_column table_name, :discarded_at, :datetime
      add_index table_name, :discarded_at, algorithm: :concurrently
    end

    safety_assured do
      remove_column :profiles, :deleted_at # standard:disable Rails/ReversibleMigration
    end
  end
end
