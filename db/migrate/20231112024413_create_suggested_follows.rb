# frozen_string_literal: true

class CreateSuggestedFollows < ActiveRecord::Migration[7.0]
  def change
    create_table :suggested_follows, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :source_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.references :target_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.timestamp :checked_at
      t.timestamps

      t.index %i[source_profile_id target_profile_id], name: :index_suggested_follows_on_src_profile_id_and_tgt_profile_id, unique: true
    end
  end
end
