# typed: strict
# frozen_string_literal: true

class StampRecord < ApplicationRecord
  self.table_name = "stamps"

  include ModelConcerns::Notifiable

  belongs_to :post_record, class_name: "PostRecord", foreign_key: :post_id
  belongs_to :profile_record, class_name: "ProfileRecord", foreign_key: :profile_id

  sig { void }
  def notify!
    NotificationRecord.where(
      source_profile_id: profile_id,
      target_profile_id: post_record.not_nil!.profile_id,
      notifiable: self
    ).first_or_create!(notified_at: Time.current)

    nil
  end

  sig { void }
  def unnotify!
    notification&.destroy!

    nil
  end
end
