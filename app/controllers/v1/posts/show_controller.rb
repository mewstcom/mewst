# typed: true
# frozen_string_literal: true

class V1::Posts::ShowController < V1::ApplicationController
  include PublicAuthenticatable

  def call
    post = Post.kept.find_by(id: params[:post_id])

    if post.nil?
      resource = V1::ResponseErrorResource.new(code: V1::ResponseErrorCode::NotFound, message: "Not found")
      return render(
        json: V1::ResponseErrorSerializer.new([resource]),
        status: :not_found
      )
    end

    resource = V1::PostResource.new(post:, viewer: current_viewer!)
    render(
      json: V1::PostSerializer.new(resource)
    )
  end
end
