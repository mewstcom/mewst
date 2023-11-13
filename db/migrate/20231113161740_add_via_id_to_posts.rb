# frozen_string_literal: true

class AddViaIdToPosts < ActiveRecord::Migration[7.0]
  disable_ddl_transaction!

  def change
    add_column :posts, :via_id, :uuid
    add_index :posts, :via_id, algorithm: :concurrently
    add_foreign_key :posts, :oauth_applications, column: :via_id, validate: false
  end
end
