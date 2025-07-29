# typed: strict
# frozen_string_literal: true

class DeletePostUseCase < ApplicationUseCase
  sig { params(target_post: PostRecord).void }
  def call(target_post:)
    ActiveRecord::Base.transaction do
      target_post.stamps.find_each do |stamp|
        stamp.notification&.destroy!
        stamp.reload.destroy!
      end

      target_post.home_timeline_posts.find_each(&:destroy!)
      target_post.post_link&.destroy!
      target_post.reload.destroy!
    end

    nil
  end
end
