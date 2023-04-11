# typed: true
# frozen_string_literal: true

class Api::Internal::CommentedPosts::CreateController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    comment = T.cast(params[:comment], T.nilable(String))

    ActiveRecord::Base.transaction do
      @post = current_profile!.create_commented_post(comment:)
    end

    render(status: :created)
  rescue ActiveRecord::RecordInvalid => e
    render(json: {errors: e.record.errors.full_messages}, status: :unprocessable_entity)
  end
end
