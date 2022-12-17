# typed: true
# frozen_string_literal: true

class Api::Internal::Follow::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(idname: params[:idname])

    if current_profile!.following?(target_profile:)
      return render(json: {}, status: :ok)
    end

    follow_command = Commands::FollowProfile.new(source_profile: current_profile!, target_profile:)

    if follow_command.invalid?
      return render(json: {errors: follow_command.errors.full_messages}, status: :unprocessable_entity)
    end

    follow_command.call

    render(json: {}, status: :created)
  end
end
