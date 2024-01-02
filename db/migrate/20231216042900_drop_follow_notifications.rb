# frozen_string_literal: true

class DropFollowNotifications < ActiveRecord::Migration[7.1]
  def change
    drop_table :follow_notifications # standard:disable Rails/ReversibleMigration
  end
end
