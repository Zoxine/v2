import base64


class ContentManager:
    def __init__(self):
        self.default_supersub_title = "Wm94aW5lQGdpdGh1Yi5pby92MiB8IFN1cGVyU3ViCg=="

        self.default_v2ray_title = "Wm94aW5lQGdpdGh1Yi5pby92MiB8IGFsbCDwn6aVCg=="
        self.default_v2ray_sub_title = "4pu177iPIHpveGluLmdpdGh1Yi5pby92MiB8IHN1YiUK"

        self.default_warp_title = "8J+XvSBab3hpbi5naXRodWIuaW8vdjIgfCB3YXJwIPCfjLEK"

        self.filter_titles = {
            "vmess": "8J+QiCBab3hpbmUuZ2l0aHViLmlvL3YyIHwgdm1lc3Mg8J+Yjw==",
            "vless": "8J+mlSB6b3hpbmUuZ2l0aHViLmlvL3YyIHwgdmxlc3Mg8J+Yjw==",
            "trojan": "8J+QjiBab3hpbi5naXRodWIuaW8vdjIgfCB0cm9qYW4g8J+Yjw==",
            "ss": "8J+QhSB6b3hpbmUuZ2l0aHViLmlvL3YyIHwgc3Mg8J+Yjw==",
            "ssr": "8J+QhSB6b3hpbmUuZ2l0aHViLmlvL3YyIHwgc3NyIPCfmI8=",
            "tuic": "8J+QsyBab3hpbmUuZ2l0aHViLmlvL3YyIHwgdHVpYyDwn5iP",
            "hy2": "PDAwMDFmOTllPiBab3hpbmUuZ2l0aHViLmlvL3YyIHwgaHkyIPCfmI8=",
            "hysteria2": "8J+mniBab3hpbmUuZ2l0aHViLmlvL3YyIHwgaHkyIPCfmI8="
        }

    @staticmethod
    def __get_file(file_path: str, title: str = None, default: str = None) -> str:
        with open(file_path, 'r', encoding="utf-8") as file:
            content = file.read()
            if title:
                content = content.replace('%TITLE%', base64.b64encode(title.encode()).decode())
            elif default:
                content = content.replace('%TITLE%', default)
        return content

    def get_warp(self, title: str = None) -> str:
        return self.__get_file(f'src/contents/fixed-warp',
                               title, self.default_warp_title)

    def get_filtered(self, title: str = None, protocol: str = None) -> str:
        return self.__get_file(f'src/contents/fixed-filtered',
                               title, self.filter_titles.get(protocol) if protocol else self.default_v2ray_title)

    def get_v2ray(self, title: str = None) -> str:
        return self.__get_file(f'src/contents/fixed-v2ray',
                               title, self.default_v2ray_title)
    
    def get_v2ray_supersub(self, title: str = None) -> str:
        return self.__get_file(f'src/contents/fixed-v2ray-supersub',
                               title, self.default_supersub_title)

    def get_v2ray_sub(self, sub_id: int) -> str:
        title = str(base64.b64decode(self.default_v2ray_sub_title).decode() + str(sub_id))
        return self.get_v2ray(title)
