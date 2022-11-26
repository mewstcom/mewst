# typed: true
# frozen_string_literal: true

class Api::Internal::Follow::Toggle::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(idname: params[:idname])
    source_profile = current_user.profile

    result = if current_user.following?(target_profile:)
      UnfollowProfileService.new(source_profile:, target_profile:).call
    else
      FollowProfileService.new(source_profile:, target_profile:).call
    end

    if result.errors.any?
      render(json: { errors: result.errors }, status: :unprocessable_entity)
    else
      render(json: {}, status: :ok)
    end
  end
end
