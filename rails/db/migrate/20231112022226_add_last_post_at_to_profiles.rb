# frozen_string_literal: true

class AddLastPostAtToProfiles < ActiveRecord::Migration[7.0]
  disable_ddl_transaction!

  def change
    add_column :profiles, :last_post_at, :datetime
    add_index :profiles, :last_post_at, algorithm: :concurrently
  end
end
