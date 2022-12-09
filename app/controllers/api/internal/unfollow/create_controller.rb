# typed: true
# frozen_string_literal: true

class Api::Internal::Unfollow::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(idname: params[:idname])

    unless T.must(current_profile).following?(target_profile:)
      return render(json: {}, status: :ok)
    end

    unfollow_creator = T.must(current_profile).unfollow(target_profile:)

    if unfollow_creator.valid?
      return render(json: {}, status: :created)
    end

    render(json: {errors: unfollow_creator.errors.full_messages}, status: :unprocessable_entity)
  end
end
