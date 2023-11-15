# frozen_string_literal: true

class AddViaIdToPosts < ActiveRecord::Migration[7.0]
  disable_ddl_transaction!

  def change
    add_column :posts, :oauth_application_id, :uuid
    add_index :posts, :oauth_application_id, algorithm: :concurrently
    add_foreign_key :posts, :oauth_applications, column: :oauth_application_id, validate: false
  end
end
