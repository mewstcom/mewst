# typed: strict
# frozen_string_literal: true

class Stamp < ApplicationRecord
  include ModelConcerns::Notifiable

  belongs_to :post
  belongs_to :profile

  sig { void }
  def notify!
    Notification.where(
      source_profile: profile,
      target_profile: post.not_nil!.profile,
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
