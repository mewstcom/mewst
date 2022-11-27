# typed: true
# frozen_string_literal: true

class Api::Internal::Follow::Toggle::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    form = FollowForm.new(source_idname: T.must(current_user).idname, target_idname: params[:idname])

    if form.invalid?
      return render(json: { errors: form.errors.full_messages }, status: :unprocessable_entity)
    end

    ActiveRecord::Base.transaction do
      if T.must(current_user).following?(target_profile: T.must(form.target_profile))
        UnfollowProfileService.new(form:).call
      else
        FollowProfileService.new(form:).call
      end
    end

    render(json: {}, status: :ok)
  end
end
