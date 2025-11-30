# frozen_string_literal: true

class FixNotificationsForDelegatedType < ActiveRecord::Migration[7.1]
  def change
    add_column :notifications, :notifiable_id, :uuid, null: false
    drop_table :stamp_notifications # rubocop:disable Rails/ReversibleMigration
  end
end
