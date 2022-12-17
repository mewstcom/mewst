# typed: true
# frozen_string_literal: true

class Api::Internal::Unfollow::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(idname: params[:idname])

    unless current_profile!.following?(target_profile:)
      return render(json: {}, status: :ok)
    end

    unfollow_command = Commands::UnfollowProfile.new(source_profile: current_profile!, target_profile:)

    if unfollow_command.invalid?
      return render(json: {errors: unfollow_command.errors.full_messages}, status: :unprocessable_entity)
    end

    unfollow_command.call

    render(json: {}, status: :created)
  end
end
