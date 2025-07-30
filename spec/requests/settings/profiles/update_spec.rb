# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /settings/profile", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    patch "/settings/profile", params: {
      profile_form: {
        atname: "newatname",
        name: "新しい名前",
        description: "新しい説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、atnameが空の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "",
        name: "名前",
        description: "説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("アットネーム")
  end

  it "ログインしているとき、atnameが無効な形式の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "invalid-atname!", # 特殊文字を含む
        name: "名前",
        description: "説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("アットネーム")
  end

  it "ログインしているとき、atnameが30文字を超える場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "a" * 31,
        name: "名前",
        description: "説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("アットネーム")
  end

  it "ログインしているとき、atnameが既に使用されている場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    FactoryBot.create(:profile_record, atname: "taken")
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "taken",
        name: "名前",
        description: "説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("アットネーム")
  end

  it "ログインしているとき、avatar_kindが無効な値の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "newatname",
        name: "名前",
        description: "説明",
        avatar_kind: "invalid",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("アバター")
  end

  it "ログインしているとき、avatar_kindがgravatarでgravatar_emailが空の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "newatname",
        name: "名前",
        description: "説明",
        avatar_kind: "gravatar",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("Gravatar")
  end

  it "ログインしているとき、avatar_kindがexternalでimage_urlが空の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "newatname",
        name: "名前",
        description: "説明",
        avatar_kind: "external",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("画像URL")
  end

  it "ログインしているとき、image_urlが無効なURL形式の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "newatname",
        name: "名前",
        description: "説明",
        avatar_kind: "external",
        gravatar_email: "",
        image_url: "not-a-url"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("画像URL")
  end

  it "ログインしているとき、有効なパラメータでプロフィールが更新されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "newatname",
        name: "新しい名前",
        description: "新しい説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to redirect_to(settings_profile_path)
    expect(flash[:notice]).to eq("プロフィールを更新しました")

    actor.profile_record.reload
    expect(actor.profile_record.atname).to eq("newatname")
    expect(actor.profile_record.name).to eq("新しい名前")
    expect(actor.profile_record.description).to eq("新しい説明")
  end

  it "ログインしているとき、avatar_kindがgravatarの場合、gravatar_emailが保存されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: actor.atname,
        name: "名前",
        description: "説明",
        avatar_kind: "gravatar",
        gravatar_email: "gravatar@example.com",
        image_url: ""
      }
    }

    expect(response).to redirect_to(settings_profile_path)

    actor.profile_record.reload
    expect(actor.profile_record.avatar_kind).to eq("gravatar")
    expect(actor.profile_record.gravatar_email).to eq("gravatar@example.com")
  end

  it "ログインしているとき、avatar_kindがexternalの場合、image_urlが保存されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: actor.atname,
        name: "名前",
        description: "説明",
        avatar_kind: "external",
        gravatar_email: "",
        image_url: "https://example.com/avatar.png"
      }
    }

    expect(response).to redirect_to(settings_profile_path)

    actor.profile_record.reload
    expect(actor.profile_record.avatar_kind).to eq("external")
    expect(actor.profile_record.image_url).to eq("https://example.com/avatar.png")
  end

  it "ログインしているとき、自分のatnameを再度設定してもエラーにならないこと" do
    actor = FactoryBot.create(:actor)
    actor.profile_record.update!(atname: "myatname")
    sign_in(actor)

    patch "/settings/profile", params: {
      profile_form: {
        atname: "myatname", # 同じatname
        name: "新しい名前",
        description: "新しい説明",
        avatar_kind: "default",
        gravatar_email: "",
        image_url: ""
      }
    }

    expect(response).to redirect_to(settings_profile_path)
    expect(flash[:notice]).to eq("プロフィールを更新しました")
  end
end
