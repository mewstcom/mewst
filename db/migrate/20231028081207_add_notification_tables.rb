# frozen_string_literal: true

class AddNotificationTables < ActiveRecord::Migration[7.0]
  def change
    create_table :notifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.string :notifiable_type, null: false
      t.timestamp :notified_at, index: true, null: false
      t.timestamp :read_at
      t.timestamps
    end

    create_table :follow_notifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :notification, foreign_key: true, null: false, type: :uuid
      t.references :source_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.references :target_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.timestamps

      t.index %i[source_profile_id target_profile_id], name: :index_follow_notifications_on_src_profile_id_and_tgt_profile_id, unique: true
    end

    create_table :stamp_notifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :notification, foreign_key: true, null: false, type: :uuid
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.references :post, foreign_key: true, null: false, type: :uuid
      t.timestamps

      t.index %i[profile_id post_id], unique: true
    end
  end
end
