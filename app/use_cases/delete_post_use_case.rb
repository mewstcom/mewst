# typed: strict
# frozen_string_literal: true

class DeletePostUseCase < ApplicationUseCase
  sig { params(target_post: PostRecord).void }
  def call(target_post:)
    ActiveRecord::Base.transaction do
      target_post.stamp_records.find_each do |stamp|
        stamp.notification_record&.destroy!
        stamp.reload.destroy!
      end

      target_post.home_timeline_post_records.find_each(&:destroy!)
      target_post.post_link_record&.destroy!
      target_post.reload.destroy!
    end

    nil
  end
end
